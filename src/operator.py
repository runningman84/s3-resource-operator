"""S3 Resource Operator - main orchestration logic."""

import logging
import os
import threading
import time
from kubernetes import watch
from prometheus_client import Counter, Histogram
from typing import Any, Dict

logger = logging.getLogger("s3-resource-operator")

# Prometheus metrics
SECRETS_PROCESSED = Counter(
    's3_operator_secrets_processed_total', 'Total number of secrets processed')
ERRORS_TOTAL = Counter('s3_operator_errors_total',
                       'Total number of errors encountered')
SYNC_DURATION = Histogram(
    's3_operator_sync_duration_seconds', 'Duration of a sync cycle')
HANDLE_SECRET_DURATION = Histogram(
    's3_operator_handle_secret_duration_seconds', 'Duration of handling a secret')

# Resource operation metrics
USERS_CREATED = Counter('s3_operator_users_created_total',
                        'Total number of users created')
USERS_DELETED = Counter('s3_operator_users_deleted_total',
                        'Total number of users deleted')
USERS_UPDATED = Counter('s3_operator_users_updated_total',
                        'Total number of users updated')
BUCKETS_CREATED = Counter(
    's3_operator_buckets_created_total', 'Total number of buckets created')
BUCKET_OWNERS_CHANGED = Counter(
    's3_operator_bucket_owners_changed_total', 'Total number of bucket owners changed')


class Operator:
    """Main operator that watches secrets and manages S3 resources."""

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
        """Main entry point for the operator."""
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
        missing_fields = [
            field for field in required_fields if not secret_data.get(field)]
        if missing_fields:
            ERRORS_TOTAL.inc()
            raise Exception(
                f"Secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}' is missing required fields: {', '.join(missing_fields)}")

        bucket_name = secret_data.get('bucket-name')
        access_key = secret_data.get('access-key')
        secret_key = secret_data.get('access-secret')
        endpoint_url = secret_data.get('endpoint-url')

        if endpoint_url and endpoint_url != self.backend.endpoint_url:
            ERRORS_TOTAL.inc()
            raise Exception(
                f"Endpoint URL {endpoint_url} in secret '{secret.metadata.name}' does not match operator configuration {self.backend.endpoint_url}.")

        logger.info(
            f"Processing bucket '{bucket_name}' and user '{access_key}'.")

        if not self.backend.bucket_exists(bucket_name):
            logger.info(f"Creating S3 bucket '{bucket_name}'.")
            self.backend.create_bucket(bucket_name, owner=access_key)
            BUCKETS_CREATED.inc()
        else:
            logger.info(f"S3 bucket '{bucket_name}' already exists.")
            owner = self.backend.get_bucket_owner(bucket_name)
            if not owner:
                logger.warning(
                    f"Could not determine owner of bucket '{bucket_name}'.")
            if owner and owner != access_key:
                logger.info(
                    f"Changing owner of bucket '{bucket_name}' to '{access_key}'.")
                self.backend.change_bucket_owner(bucket_name, access_key)
                BUCKET_OWNERS_CHANGED.inc()
            else:
                logger.info(
                    f"S3 bucket '{bucket_name}' is already owned by '{access_key}'.")

        if not self.backend.user_exists(access_key):
            logger.info(f"Creating IAM user '{access_key}'.")
            self.backend.create_user(
                access_key=access_key,
                secret_key=secret_key,
                role=secret_data.get('role'),
                user_id=secret_data.get('user-id'),
                group_id=secret_data.get('group-id')
            )
            USERS_CREATED.inc()
        else:
            logger.info(
                f"IAM user '{access_key}' already exists. Skipping creation.")

        SECRETS_PROCESSED.inc()

    @SYNC_DURATION.time()
    def sync(self):
        """
        The main reconciliation loop.
        """
        logger.info("Starting sync cycle...")
        for secret in self.secret_manager.find_secrets():
            try:
                logger.info(
                    f"Handling secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}'")
                self.handle_secret(secret)
            except Exception as e:
                ERRORS_TOTAL.inc()
                logger.error(
                    f"Error handling secret '{secret.metadata.name}': {e}")
        logger.info("Sync cycle finished.")

    def watch(self):
        """Watch for changes to secrets and handle them."""
        w = watch.Watch()
        try:
            while not self._shutdown.is_set():
                try:
                    for event in w.stream(self.v1_api.list_secret_for_all_namespaces, timeout_seconds=10):
                        if self._shutdown.is_set():
                            logger.info(
                                "Shutdown requested during watch loop. Exiting...")
                            break

                        # Type assertion: event is always a dict from Kubernetes watch
                        # type: ignore[assignment]
                        event_dict: Dict[str, Any] = event

                        secret = event_dict.get('object')
                        if secret is None:
                            continue

                        # Check annotation first before processing
                        if not (secret.metadata.annotations and self.secret_manager.annotation_key in secret.metadata.annotations):
                            continue

                        event_type = event_dict.get('type', 'UNKNOWN')
                        logger.info(
                            f"Handling '{event_type}' event for secret '{secret.metadata.name}' in ns '{secret.metadata.namespace}'")

                        if event_type in ['ADDED', 'MODIFIED']:
                            try:
                                self.handle_secret(secret)
                            except Exception as e:
                                ERRORS_TOTAL.inc()
                                logger.error(
                                    f"Error handling secret '{secret.metadata.name}': {e}")
                except Exception as e:
                    if self._shutdown.is_set():
                        break
                    logger.warning(
                        f"Watch stream interrupted, will reconnect: {e}")
                    time.sleep(1)  # Brief pause before reconnecting
        finally:
            w.stop()
            logger.info("Watch loop stopped.")
