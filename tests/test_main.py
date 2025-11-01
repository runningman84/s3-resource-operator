import pytest
from unittest.mock import MagicMock, patch
from src.main import Operator, SecretManager
import base64

def create_mock_secret(name, namespace, annotations, data):
    """Helper function to create a mock Kubernetes secret."""
    secret = MagicMock()
    secret.metadata.name = name
    secret.metadata.namespace = namespace
    secret.metadata.annotations = annotations
    secret.data = {k: base64.b64encode(v.encode('utf-8')).decode('utf-8') for k, v in data.items()}
    return secret

def test_find_secrets(kube_client):
    """Test finding secrets with the correct annotation."""
    annotation_key = "s3-resource-operator/enabled"
    
    # Create mock secrets
    secret1 = create_mock_secret("secret1", "default", {annotation_key: "true"}, {})
    secret2 = create_mock_secret("secret2", "default", {}, {})
    secret3 = create_mock_secret("secret3", "kube-system", {annotation_key: "true"}, {})

    # Configure the mock client to return these secrets
    kube_client.list_secret_for_all_namespaces.return_value.items = [secret1, secret2, secret3]

    # Initialize the SecretManager and find secrets
    manager = SecretManager(kube_client, annotation_key)
    found_secrets = manager.find_secrets()

    # Assert that only secrets with the annotation are found
    assert len(found_secrets) == 2
    assert secret1 in found_secrets
    assert secret3 in found_secrets

def test_process_secret(kube_client):
    """Test processing a secret to decode its data."""
    annotation_key = "s3-resource-operator/enabled"
    
    # Secret data
    data = {
        "bucket-name": "test-bucket",
        "access-key": "test-user",
        "access-secret": "test-password"
    }

    # Create a mock secret
    secret = create_mock_secret("secret1", "default", {annotation_key: "true"}, data)

    # Initialize the SecretManager and process the secret
    manager = SecretManager(kube_client, annotation_key)
    processed_data = manager.process_secret(secret)

    # Assert that the data is correctly decoded
    assert processed_data == data

def test_handle_secret_creation(versitygw_client):
    """Test handling a secret for a new bucket and user."""
    # Mock backend and secret manager
    backend = versitygw_client
    secret_manager = MagicMock()

    # Operator instance
    op = Operator(backend, secret_manager)

    # Mock secret data
    secret_data = {
        "bucket-name": "new-bucket",
        "access-key": "new-user",
        "access-secret": "new-password"
    }
    secret_manager.process_secret.return_value = secret_data

    # Mock backend methods
    backend.bucket_exists.return_value = False
    backend.user_exists.return_value = False

    # Create a mock secret object
    mock_secret = create_mock_secret("new-secret", "default", {}, secret_data)

    # Handle the secret
    op.handle_secret(mock_secret)

    # Assertions
    backend.bucket_exists.assert_called_once_with("new-bucket")
    backend.create_bucket.assert_called_once_with("new-bucket", owner="new-user")
    backend.user_exists.assert_called_once_with("new-user")
    backend.create_user.assert_called_once()

def test_handle_secret_existing_resources(versitygw_client):
    """Test handling a secret for existing resources."""
    backend = versitygw_client
    secret_manager = MagicMock()
    op = Operator(backend, secret_manager)

    secret_data = {
        "bucket-name": "existing-bucket",
        "access-key": "existing-user",
        "access-secret": "existing-password"
    }
    secret_manager.process_secret.return_value = secret_data

    backend.bucket_exists.return_value = True
    backend.user_exists.return_value = True
    backend.get_bucket_owner.return_value = "existing-user"

    mock_secret = create_mock_secret("existing-secret", "default", {}, secret_data)

    op.handle_secret(mock_secret)

    backend.create_bucket.assert_not_called()
    backend.create_user.assert_not_called()
    backend.change_bucket_owner.assert_not_called()

def test_handle_secret_change_owner(versitygw_client):
    """Test changing the owner of an existing bucket."""
    backend = versitygw_client
    secret_manager = MagicMock()
    op = Operator(backend, secret_manager)

    secret_data = {
        "bucket-name": "existing-bucket",
        "access-key": "new-owner",
        "access-secret": "new-password"
    }
    secret_manager.process_secret.return_value = secret_data

    backend.bucket_exists.return_value = True
    backend.get_bucket_owner.return_value = "old-owner"

    mock_secret = create_mock_secret("owner-change-secret", "default", {}, secret_data)

    op.handle_secret(mock_secret)

    backend.change_bucket_owner.assert_called_once_with("existing-bucket", "new-owner")
