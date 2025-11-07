import logging
from .backend import Backend

logger = logging.getLogger("s3-resource-operator.minio")


class Minio(Backend):
    """
    Dummy backend for MinIO.
    This is a placeholder and does not implement any real functionality.
    """

    def __init__(self, endpoint_url, access_key, secret_key):
        self.endpoint_url = endpoint_url
        self.access_key = access_key
        self.secret_key = secret_key
        logger.info("Initialized MinIO dummy backend.")

    def test_connection(self):
        logger.info(f"Testing connection to backend: {self.endpoint_url}")
        logger.info(f"Using access key: {self.access_key}")
        logger.info("[MinIO DUMMY] Would test connection.")
        raise NotImplementedError("MinIO backend is not fully implemented")

    def create_bucket(self, bucket_name):
        logger.info(f"[MinIO DUMMY] Would create bucket '{bucket_name}'.")
        raise NotImplementedError

    def delete_bucket(self, bucket_name):
        logger.info(f"[MinIO DUMMY] Would delete bucket '{bucket_name}'.")
        raise NotImplementedError

    def list_buckets(self):
        logger.info("[MinIO DUMMY] Would list buckets.")
        raise NotImplementedError

    def bucket_exists(self, bucket_name):
        logger.info(
            f"[MinIO DUMMY] Would check if bucket '{bucket_name}' exists.")
        raise NotImplementedError

    def create_user(self, access_key, secret_key, **kwargs):
        logger.info(f"[MinIO DUMMY] Would create user '{access_key}'.")
        raise NotImplementedError

    def delete_user(self, access_key):
        logger.info(f"[MinIO DUMMY] Would delete user '{access_key}'.")
        raise NotImplementedError

    def list_users(self):
        logger.info("[MinIO DUMMY] Would list users.")
        raise NotImplementedError

    def user_exists(self, access_key):
        logger.info(
            f"[MinIO DUMMY] Would check if user '{access_key}' exists.")
        raise NotImplementedError

    def change_bucket_owner(self, bucket_name, new_owner):
        logger.info(
            f"[MinIO DUMMY] Would change owner of bucket '{bucket_name}' to '{new_owner}'.")
        raise NotImplementedError

    def owner_exists(self, bucket_name, username):
        logger.info(
            f"[MinIO DUMMY] Would check if '{username}' owns '{bucket_name}'.")
        raise NotImplementedError

    def get_bucket_owner(self, bucket_name):
        logger.info(
            f"[MinIO DUMMY] Would get owner of bucket '{bucket_name}'.")
        raise NotImplementedError

    def update_user(self, access_key, secret_key=None, user_id=None, group_id=None):
        logger.info(f"[MinIO DUMMY] Would update user '{access_key}'.")
        raise NotImplementedError
