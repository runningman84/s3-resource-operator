"""Tests for SecretManager class."""

import pytest
from unittest.mock import MagicMock
from src.secrets import SecretManager
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


def test_find_secrets(kube_client):
    """Test finding secrets with the correct annotation."""
    annotation_key = "s3-resource-operator/enabled"

    # Create mock secrets
    secret1 = create_mock_secret("secret1", "default", {
                                 annotation_key: "true"}, {})
    secret2 = create_mock_secret("secret2", "default", {}, {})
    secret3 = create_mock_secret(
        "secret3", "kube-system", {annotation_key: "true"}, {})

    # Configure the mock client to return these secrets
    kube_client.list_secret_for_all_namespaces.return_value.items = [
        secret1, secret2, secret3]

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
        "secret-key": "test-password",
        "endpoint-url": "http://s3.example.com"
    }

    # Create a mock secret
    secret = create_mock_secret("secret1", "default", {
                                annotation_key: "true"}, data)

    # Initialize the SecretManager and process the secret
    manager = SecretManager(kube_client, annotation_key)
    processed_data = manager.process_secret(secret)

    # Assert that the data is correctly decoded
    assert processed_data == data
