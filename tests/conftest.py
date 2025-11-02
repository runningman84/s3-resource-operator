import pytest
from unittest.mock import MagicMock, patch
from prometheus_client import REGISTRY

@pytest.fixture
def kube_client():
    """Fixture for a mock Kubernetes client."""
    return MagicMock()

@pytest.fixture
def s3_client():
    """Fixture for a mock S3 client."""
    with patch('boto3.client') as mock_boto3_client:
        yield mock_boto3_client

@pytest.fixture
def operator_kwargs(kube_client):
    """Fixture for operator keyword arguments."""
    return {
        "group": "s3-resource-operator.io",
        "version": "v1",
        "plural": "s3resources",
        "namespace": "default",
        "kube_client": kube_client,
    }

@pytest.fixture
def versitygw_client():
    """Fixture for a mock VersityGW client."""
    return MagicMock()

@pytest.fixture(autouse=True)
def clear_prometheus_registry():
    """Clear the default Prometheus registry before each test."""
    collectors = list(REGISTRY._collector_to_names.keys())
    for collector in collectors:
        REGISTRY.unregister(collector)
