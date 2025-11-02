import logging
from kubernetes import client, config, watch
import os
import base64

from .backends import get_backend

logger = logging.getLogger("s3-resource-operator")

def get_k8s_api():
    # Load kube config
    try:
        config.load_incluster_config()
        logger.info("Loaded in-cluster kube config.")
    except config.ConfigException:
        logger.info("Could not load in-cluster config. Falling back to local kube config.")
        config.load_kube_config()
    return client.CoreV1Api()

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

    def handle_secret(self, secret):
        """
        Processes a single secret to ensure the corresponding bucket and user exist
        and are correctly configured.
        """
        secret_data = self.secret_manager.process_secret(secret)

        required_fields = ['bucket-name', 'access-key', 'access-secret', 'endpoint-url']
        missing_fields = [field for field in required_fields if not secret_data.get(field)]
        if missing_fields:
            raise Exception(f"Secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}' is missing required fields: {', '.join(missing_fields)}")

        bucket_name = secret_data.get('bucket-name')
        access_key = secret_data.get('access-key')
        secret_key = secret_data.get('access-secret')
        endpoint_url = secret_data.get('endpoint-url')

        if endpoint_url and endpoint_url != self.backend.endpoint_url:
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

    def sync(self):
        logger.info("Starting sync cycle...")
        for secret in self.secret_manager.find_secrets():
            try:
                logger.info(f"Handling secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}'")
                self.handle_secret(secret)
            except Exception as e:
                logger.error(f"Error handling secret '{secret.metadata.name}': {e}")
        logger.info("Sync cycle finished.")

    def run(self):
        w = watch.Watch()
        for event in w.stream(self.v1_api.list_secret_for_all_namespaces):
            secret = event['object']
            event_type = event['type']

            if not (secret.metadata.annotations and self.secret_manager.annotation_key in secret.metadata.annotations):
                continue

            logger.info(f"Handling '{event_type}' event for secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}'")

            if event_type in ['ADDED', 'MODIFIED']:
                try:
                    self.handle_secret(secret)
                except Exception as e:
                    logger.error(f"Error handling secret '{secret.metadata.name}': {e}")

# --- Main Execution ---
if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    annotation_key = os.environ.get("ANNOTATION_KEY", "s3-resource-operator/enabled")
    s3_endpoint_url = os.environ.get("S3_ENDPOINT_URL")
    s3_access_key = os.environ.get("S3_ACCESS_KEY")
    s3_secret_key = os.environ.get("S3_SECRET_KEY")
    backend_name = os.environ.get("BACKEND_NAME", "versitygw")

    required_env_vars = {
        "S3_ENDPOINT_URL": s3_endpoint_url,
        "S3_ACCESS_KEY": s3_access_key,
        "S3_SECRET_KEY": s3_secret_key,
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

    logger.info(f"Starting inital sync...")

    operator.sync()

    logger.info("Starting continuous run...")

    operator.run()