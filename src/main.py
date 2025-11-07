"""S3 Resource Operator - Entry point."""

import logging
import os
import signal
import sys
from dotenv import load_dotenv

from .backends import get_backend
from .metrics import MetricsServer
from .secrets import SecretManager
from .operator import Operator
from .utils import get_k8s_api

logger = logging.getLogger("s3-resource-operator")


def main():
    """Main entry point for the S3 Resource Operator."""
    # Load environment variables from .env file
    load_dotenv()

    # Get log level from environment variable (default to INFO)
    log_level = os.environ.get("LOG_LEVEL", "INFO").upper()
    valid_log_levels = ["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]
    if log_level not in valid_log_levels:
        log_level = "INFO"

    # Configure logging
    logging.basicConfig(
        level=getattr(logging, log_level),
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    # Start up the server to expose the metrics.
    metrics_server = MetricsServer(port=8000)
    metrics_server.start()

    logger.info("Prometheus metrics server started on port 8000.")
    logger.info(f"Log level set to: {log_level}")

    annotation_key = os.environ.get(
        "ANNOTATION_KEY", "s3-resource-operator.io/enabled")
    s3_endpoint_url = os.environ.get("S3_ENDPOINT_URL")
    s3_access_key = os.environ.get("ROOT_ACCESS_KEY")
    s3_secret_key = os.environ.get("ROOT_SECRET_KEY")
    backend_name = os.environ.get("BACKEND_NAME", "versitygw")

    required_env_vars = {
        "ANNOTATION_KEY": annotation_key,
        "S3_ENDPOINT_URL": s3_endpoint_url,
        "ROOT_ACCESS_KEY": s3_access_key,
        "ROOT_SECRET_KEY": s3_secret_key,
        "BACKEND_NAME": backend_name
    }
    missing_vars = [key for key,
                    value in required_env_vars.items() if not value]

    if missing_vars:
        logger.error(
            f"Missing required environment variables: {', '.join(missing_vars)}")
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

    logger.info("Testing backend connection...")
    try:
        backend.test_connection()
    except Exception as e:
        logger.error(f"Backend connection test failed: {e}")
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
