import abc

class Backend(abc.ABC):
    """Abstract base class for a storage backend."""

    @abc.abstractmethod
    def create_bucket(self, bucket_name, owner=None):
        """Create a bucket."""
        pass

    @abc.abstractmethod
    def delete_bucket(self, bucket_name):
        """Delete a bucket."""
        pass

    @abc.abstractmethod
    def bucket_exists(self, bucket_name):
        """Check if a bucket exists."""
        pass

    @abc.abstractmethod
    def get_bucket_owner(self, bucket_name):
        """Get the owner of a bucket."""
        pass

    @abc.abstractmethod
    def change_bucket_owner(self, bucket_name, new_owner):
        """Change the owner of a bucket."""
        pass

    @abc.abstractmethod
    def create_user(self, access_key, secret_key, role=None, user_id=None, group_id=None):
        """Create a user."""
        pass

    @abc.abstractmethod
    def delete_user(self, access_key):
        """Delete a user."""
        pass

    @abc.abstractmethod
    def update_user(self, access_key, secret_key=None, user_id=None, group_id=None):
        """Update a user."""
        pass

    @abc.abstractmethod
    def user_exists(self, access_key):
        """Check if a user exists."""
        pass