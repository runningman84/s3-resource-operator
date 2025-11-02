import logging
from kubernetes import client, config, watch
import os
import base64
from prometheus_client import start_http_server, Counter, Histogram, generate_latest
import time
from dotenv import load_dotenv
from http.server import BaseHTTPRequestHandler, HTTPServer
import threading
import signal
import sys

from .backends import get_backend

logger = logging.getLogger("s3-resource-operator")

# Prometheus metrics
SECRETS_PROCESSED = Counter('s3_operator_secrets_processed_total', 'Total number of secrets processed')
ERRORS_TOTAL = Counter('s3_operator_errors_total', 'Total number of errors encountered')
SYNC_DURATION = Histogram('s3_operator_sync_duration_seconds', 'Duration of a sync cycle')
HANDLE_SECRET_DURATION = Histogram('s3_operator_handle_secret_duration_seconds', 'Duration of handling a secret')


def get_k8s_api():
    # Load kube config
    try:
        config.load_incluster_config()
        logger.info("Loaded in-cluster kube config.")
    except config.ConfigException:
        logger.info("Could not load in-cluster config. Falling back to local kube config.")
        config.load_kube_config()
    return client.CoreV1Api()

class MetricsServer:
    def __init__(self, port=8000):
        self.port = port
        self.server = HTTPServer(('', self.port), self._get_handler())

    def _get_handler(self):
        class MetricsHandler(BaseHTTPRequestHandler):
            def do_GET(self):
                if self.path == '/metrics':
                    self.send_response(200)
                    self.send_header('Content-type', 'text/plain')
                    self.end_headers()
                    self.wfile.write(generate_latest())
                else:
                    self.send_response(404)
                    self.end_headers()
                    self.wfile.write(b"Not Found")
        return MetricsHandler

    def start(self):
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.daemon = True
        self.thread.start()
        logger.info(f"Metrics server started on port {self.port} at /metrics")

class SecretManager:
    def __init__(self, v1_api, annotation_key):
        self.v1 = v1_api
        self.annotation_key = annotation_key

    def find_secrets(self):
        secrets = self.v1.list_secret_for_all_namespaces().items
        return [s for s in secrets if self.annotation_key in (s.metadata.annotations or {})]

    def process_secret(self, secret):
        logger.debug(f"Processing secret '{secret.metadata.name}' in namespace '{secret.metadata.namespace}'")
        return {
            key: base64.b64decode(value).decode("utf-8")
            for key, value in secret.data.items()
        }

class Operator:
    def __init__(self, v1_api, backend, secret_manager):
        self.v1_api = v1_api
        self.backend = backend
        self.secret_manager = secret_manager
        self._shutdown = threading.Event()

    def shutdown(self):
        """Signal the operator to shutdown gracefully."""
        logger.info("Shutdown signal received. Stopping operator...")
        self._shutdown.set()

    def run(self):
        if os.environ.get("TEST_MODE") == "true":
            logger.info("TEST_MODE is enabled. Skipping operator run.")
            return
        
        logger.info(f"Starting inital sync...")

        self.sync()

        logger.info("Starting continuous watch...")

        self.watch()

    @HANDLE_SECRET_DURATION.time()
    def handle_secret(self, secret):
        """
        Processes a single secret to ensure the corresponding bucket and user exist
        and are correctly configured.
        """
        secret_data = self.secret_manager.process_secret(secret)

        required_fields = ['bucket-name', 'access-key', 'access-secret']
        missing_fields = [field for field in required_fields if not secret_data.get(field)]
        if missing_fields:
            ERRORS_TOTAL.inc()
            raise Exception(f"Secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}' is missing required fields: {', '.join(missing_fields)}")

        bucket_name = secret_data.get('bucket-name')
        access_key = secret_data.get('access-key')
        secret_key = secret_data.get('access-secret')
        endpoint_url = secret_data.get('endpoint-url')

        if endpoint_url and endpoint_url != self.backend.endpoint_url:
            ERRORS_TOTAL.inc()
            raise Exception(f"Endpoint URL {endpoint_url} in secret '{secret.metadata.name}' does not match operator configuration {self.backend.endpoint_url}.")

        logger.info(f"Processing bucket '{bucket_name}' and user '{access_key}'.")

        if not self.backend.bucket_exists(bucket_name):
            logger.info(f"Creating S3 bucket '{bucket_name}'.")
            self.backend.create_bucket(bucket_name, owner=access_key)
        else:
            logger.info(f"S3 bucket '{bucket_name}' already exists.")
            owner = self.backend.get_bucket_owner(bucket_name)
            if not owner:
                logger.warning(f"Could not determine owner of bucket '{bucket_name}'.")
            if owner and owner != access_key:
                logger.info(f"Changing owner of bucket '{bucket_name}' to '{access_key}'.")
                self.backend.change_bucket_owner(bucket_name, access_key)
            else:
                logger.info(f"S3 bucket '{bucket_name}' is already owned by '{access_key}'.")

        if not self.backend.user_exists(access_key):
            logger.info(f"Creating IAM user '{access_key}'.")
            self.backend.create_user(
                access_key=access_key,
                secret_key=secret_key,
                role=secret_data.get('role'),
                user_id=secret_data.get('user-id'),
                group_id=secret_data.get('group-id')
            )
        else:
            logger.info(f"IAM user '{access_key}' already exists. Skipping creation.")

        SECRETS_PROCESSED.inc()

    @SYNC_DURATION.time()
    def sync(self):
        """
        The main reconciliation loop.
        """
        logger.info("Starting sync cycle...")
        for secret in self.secret_manager.find_secrets():
            try:
                logger.info(f"Handling secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}'")
                self.handle_secret(secret)
            except Exception as e:
                ERRORS_TOTAL.inc()
                logger.error(f"Error handling secret '{secret.metadata.name}': {e}")
        logger.info("Sync cycle finished.")

    def watch(self):
        w = watch.Watch()
        try:
            while not self._shutdown.is_set():
                try:
                    for event in w.stream(self.v1_api.list_secret_for_all_namespaces, timeout_seconds=10):
                        if self._shutdown.is_set():
                            logger.info("Shutdown requested during watch loop. Exiting...")
                            break
                            
                        secret = event['object']
                        event_type = event['type']

                        if not (secret.metadata.annotations and self.secret_manager.annotation_key in secret.metadata.annotations):
                            continue

                        logger.info(f"Handling '{event_type}' event for secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}'")

                        if event_type in ['ADDED', 'MODIFIED']:
                            try:
                                self.handle_secret(secret)
                            except Exception as e:
                                ERRORS_TOTAL.inc()
                                logger.error(f"Error handling secret '{secret.metadata.name}': {e}")
                except Exception as e:
                    if self._shutdown.is_set():
                        break
                    logger.warning(f"Watch stream interrupted, will reconnect: {e}")
                    time.sleep(1)  # Brief pause before reconnecting
        finally:
            w.stop()
            logger.info("Watch loop stopped.")

def main():
    # Load environment variables from .env file
    load_dotenv()
    # Start up the server to expose the metrics.
    metrics_server = MetricsServer(port=8000)
    metrics_server.start()
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    logger.info("Prometheus metrics server started on port 8000.")

    annotation_key = os.environ.get("ANNOTATION_KEY", "s3-resource-operator.io/enabled")
    s3_endpoint_url = os.environ.get("S3_ENDPOINT_URL")
    s3_access_key = os.environ.get("S3_ACCESS_KEY")
    s3_secret_key = os.environ.get("S3_SECRET_KEY")
    backend_name = os.environ.get("BACKEND_NAME", "versitygw")

    required_env_vars = {
        "ANNOTATION_KEY": annotation_key,
        "S3_ENDPOINT_URL": s3_endpoint_url,
        "S3_ACCESS_KEY": s3_access_key,
        "S3_SECRET_KEY": s3_secret_key,
        "BACKEND_NAME": backend_name
    }
    missing_vars = [key for key, value in required_env_vars.items() if not value]

    if missing_vars:
        logger.error(f"Missing required environment variables: {', '.join(missing_vars)}")
        exit(1)

    backend_config = {
        "endpoint_url": s3_endpoint_url,
        "access_key": s3_access_key,
        "secret_key": s3_secret_key,
    }

    logger.info(f"Initializing backend '{backend_name}'...")

    try:
        backend = get_backend(backend_name, **backend_config)
    except (ValueError, NotImplementedError) as e:
        logger.error(f"Failed to initialize backend '{backend_name}': {e}")
        exit(1)

    v1_api = get_k8s_api()
    secret_manager = SecretManager(v1_api, annotation_key)
    operator = Operator(v1_api, backend, secret_manager)

    # Setup signal handlers for graceful shutdown
    def signal_handler(signum, frame):
        signal_name = 'SIGTERM' if signum == signal.SIGTERM else 'SIGINT'
        logger.info(f"Received {signal_name}, initiating graceful shutdown...")
        operator.shutdown()

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)

    logger.info("Signal handlers registered. Starting operator...")

    try:
        operator.run()
    except KeyboardInterrupt:
        logger.info("Keyboard interrupt received, shutting down...")
        operator.shutdown()
    except Exception as e:
        logger.error(f"Unexpected error: {e}", exc_info=True)
        sys.exit(1)
    
    logger.info("Operator shutdown complete.")
    sys.exit(0)


if __name__ == '__main__':
    main()
