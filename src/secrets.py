"""Secret manager for finding and processing Kubernetes secrets."""

import base64
import logging

logger = logging.getLogger("s3-resource-operator")


class SecretManager:
    """Manages discovery and processing of annotated Kubernetes secrets."""

    def __init__(self, v1_api, annotation_key):
        self.v1 = v1_api
        self.annotation_key = annotation_key

    def find_secrets(self):
        """Find all secrets with the operator's annotation."""
        secrets = self.v1.list_secret_for_all_namespaces().items
        return [s for s in secrets if self.annotation_key in (s.metadata.annotations or {})]

    def process_secret(self, secret):
        """Decode and return the data from a secret."""
        logger.debug(
            f"Processing secret '{secret.metadata.name}' in namespace '{secret.metadata.namespace}'")
        return {
            key: base64.b64decode(value).decode("utf-8")
            for key, value in secret.data.items()
        }
