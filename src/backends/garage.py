import logging
from .backend import Backend

logger = logging.getLogger("s3-resource-operator.garage")


class Garage(Backend):
    """Backend implementation for Garage."""

    def __init__(self, endpoint_url, access_key, secret_key):
        logger.info("Initializing Garage backend.")
        self.endpoint_url = endpoint_url
        self.access_key = access_key
        self.secret_key = secret_key
        logger.info("Initialized Garage dummy backend.")

    def test_connection(self):
        logger.info(f"Testing connection to backend: {self.endpoint_url}")
        logger.info(f"Using access key: {self.access_key}")
        logger.info("[Garage DUMMY] Would test connection.")
        raise NotImplementedError("Garage backend is not fully implemented")

    def create_bucket(self, bucket_name, owner=None):
        logger.info(f"[Garage DUMMY] Would create bucket '{bucket_name}'.")
        raise NotImplementedError

    def delete_bucket(self, bucket_name):
        logger.info(f"[Garage DUMMY] Would delete bucket '{bucket_name}'.")
        raise NotImplementedError

    def bucket_exists(self, bucket_name):
        logger.info(
            f"[Garage DUMMY] Would check if bucket '{bucket_name}' exists.")
        raise NotImplementedError

    def get_bucket_owner(self, bucket_name):
        logger.info(
            f"[Garage DUMMY] Would get owner of bucket '{bucket_name}'.")
        raise NotImplementedError

    def change_bucket_owner(self, bucket_name, new_owner):
        logger.info(
            f"[Garage DUMMY] Would change owner of bucket '{bucket_name}' to '{new_owner}'.")
        raise NotImplementedError

    def create_user(self, access_key, secret_key, role=None, user_id=None, group_id=None):
        logger.info(f"[Garage DUMMY] Would create user '{access_key}'.")
        raise NotImplementedError

    def delete_user(self, access_key):
        logger.info(f"[Garage DUMMY] Would delete user '{access_key}'.")
        raise NotImplementedError

    def update_user(self, access_key, secret_key=None, user_id=None, group_id=None):
        logger.info(f"[Garage DUMMY] Would update user '{access_key}'.")
        raise NotImplementedError

    def user_exists(self, access_key):
        logger.info(
            f"[Garage DUMMY] Would check if user '{access_key}' exists.")
        raise NotImplementedError
