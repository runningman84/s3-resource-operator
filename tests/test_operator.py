"""Tests for Operator class."""

import pytest
from unittest.mock import MagicMock
from src.operator import (
    Operator,
    USERS_CREATED,
    BUCKETS_CREATED,
    BUCKET_OWNERS_CHANGED
)
import base64


def create_mock_secret(name, namespace, annotations, data):
    """Helper function to create a mock Kubernetes secret."""
    secret = MagicMock()
    secret.metadata.name = name
    secret.metadata.namespace = namespace
    secret.metadata.annotations = annotations
    secret.data = {k: base64.b64encode(
        v.encode('utf-8')).decode('utf-8') for k, v in data.items()}
    return secret


def test_handle_secret_creation(kube_client, versitygw_client):
    """Test handling a secret for a new bucket and user."""
    # Mock backend and secret manager
    backend = versitygw_client
    secret_manager = MagicMock()

    # Operator instance
    op = Operator(kube_client, backend, secret_manager)

    # Mock secret data
    secret_data = {
        "bucket-name": "new-bucket",
        "access-key": "new-user",
        "access-secret": "new-password",
        "endpoint-url": "http://versitygw:8080",
    }
    secret_manager.process_secret.return_value = secret_data

    # Mock backend methods
    backend.endpoint_url = "http://versitygw:8080"
    backend.bucket_exists.return_value = False
    backend.user_exists.return_value = False

    # Create a mock secret object
    mock_secret = create_mock_secret("new-secret", "default", {}, secret_data)

    # Get initial metric values
    initial_buckets_created = BUCKETS_CREATED._value.get()
    initial_users_created = USERS_CREATED._value.get()

    # Handle the secret
    op.handle_secret(mock_secret)

    # Assertions
    backend.bucket_exists.assert_called_once_with("new-bucket")
    backend.create_bucket.assert_called_once_with(
        "new-bucket", owner="new-user")
    backend.user_exists.assert_called_once_with("new-user")
    backend.create_user.assert_called_once()

    # Verify metrics were incremented
    assert BUCKETS_CREATED._value.get() == initial_buckets_created + 1
    assert USERS_CREATED._value.get() == initial_users_created + 1


def test_handle_secret_existing_resources(kube_client, versitygw_client):
    """Test handling a secret for existing resources."""
    backend = versitygw_client
    secret_manager = MagicMock()
    op = Operator(kube_client, backend, secret_manager)

    secret_data = {
        "bucket-name": "existing-bucket",
        "access-key": "existing-user",
        "access-secret": "existing-password",
        "endpoint-url": "http://versitygw:8080",
    }
    secret_manager.process_secret.return_value = secret_data

    backend.endpoint_url = "http://versitygw:8080"
    backend.bucket_exists.return_value = True
    backend.user_exists.return_value = True
    backend.get_bucket_owner.return_value = "existing-user"

    mock_secret = create_mock_secret(
        "existing-secret", "default", {}, secret_data)

    op.handle_secret(mock_secret)

    backend.create_bucket.assert_not_called()
    backend.create_user.assert_not_called()
    backend.change_bucket_owner.assert_not_called()


def test_handle_secret_change_owner(kube_client, versitygw_client):
    """Test changing the owner of an existing bucket to an existing user."""
    backend = versitygw_client
    secret_manager = MagicMock()
    op = Operator(kube_client, backend, secret_manager)

    secret_data = {
        "bucket-name": "existing-bucket",
        "access-key": "new-owner",
        "access-secret": "new-password",
        "endpoint-url": "http://s3.example.com"
    }
    secret_manager.process_secret.return_value = secret_data

    backend.endpoint_url = "http://s3.example.com"
    backend.bucket_exists.return_value = True
    backend.user_exists.return_value = True  # User already exists
    backend.get_bucket_owner.return_value = "old-owner"

    mock_secret = create_mock_secret(
        "owner-change-secret", "default", {}, secret_data)

    # Get initial metric value
    initial_owners_changed = BUCKET_OWNERS_CHANGED._value.get()

    op.handle_secret(mock_secret)

    backend.change_bucket_owner.assert_called_once_with(
        "existing-bucket", "new-owner")
    backend.create_user.assert_not_called()  # User already exists

    # Verify metric was incremented
    assert BUCKET_OWNERS_CHANGED._value.get() == initial_owners_changed + 1


def test_handle_secret_existing_bucket_new_user(kube_client, versitygw_client):
    """Test handling a secret for an existing bucket but new user (the bug scenario)."""
    backend = versitygw_client
    secret_manager = MagicMock()
    op = Operator(kube_client, backend, secret_manager)

    secret_data = {
        "bucket-name": "existing-bucket",
        "access-key": "new-user",
        "access-secret": "new-password",
        "endpoint-url": "http://s3.example.com"
    }
    secret_manager.process_secret.return_value = secret_data

    backend.endpoint_url = "http://s3.example.com"
    backend.bucket_exists.return_value = True
    backend.user_exists.return_value = False  # User doesn't exist yet
    backend.get_bucket_owner.return_value = "old-owner"

    mock_secret = create_mock_secret(
        "bucket-new-user-secret", "default", {}, secret_data)

    # Get initial metric values
    initial_users_created = USERS_CREATED._value.get()
    initial_owners_changed = BUCKET_OWNERS_CHANGED._value.get()

    op.handle_secret(mock_secret)

    # Verify user is created BEFORE attempting to change bucket owner
    assert backend.user_exists.call_count == 1
    backend.create_user.assert_called_once()
    backend.create_bucket.assert_not_called()  # Bucket already exists
    backend.change_bucket_owner.assert_called_once_with(
        "existing-bucket", "new-user")

    # Verify call order: user_exists should be called before change_bucket_owner
    call_order = [call[0] for call in backend.method_calls]
    user_exists_index = call_order.index('user_exists')
    change_owner_index = call_order.index('change_bucket_owner')
    assert user_exists_index < change_owner_index, "user_exists must be called before change_bucket_owner"

    # Verify metrics were incremented
    assert USERS_CREATED._value.get() == initial_users_created + 1
    assert BUCKET_OWNERS_CHANGED._value.get() == initial_owners_changed + 1
