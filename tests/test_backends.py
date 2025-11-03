"""Tests for S3 backend implementations."""

import pytest
from src.backends import get_backend, SUPPORTED_BACKENDS
from src.backends.versitygw import VersityGW
from src.backends.garage import Garage
from src.backends.minio import Minio


@pytest.mark.parametrize("name, expected_class", SUPPORTED_BACKENDS.items())
def test_get_backend_supported(name, expected_class):
    """Test getting a supported backend."""
    if name == "garage":
        pytest.skip("Garage backend is not fully implemented")
    if name == "minio":
        pytest.skip("Minio backend is not fully implemented")
    backend = get_backend(name, "http://localhost", "user", "pass")
    assert isinstance(backend, expected_class)


def test_get_backend_unsupported():
    """Test getting an unsupported backend."""
    with pytest.raises(ValueError):
        get_backend("unsupported-backend", "http://localhost", "user", "pass")


def test_versitygw_init():
    """Test VersityGW backend initialization."""
    backend = VersityGW("http://versitygw", "admin", "password")
    assert backend.endpoint_url == "http://versitygw"
    assert backend.access_key == "admin"
    assert backend.secret_key == "password"


def test_garage_init():
    """Test Garage backend initialization."""
    backend = Garage("http://garage", "admin", "password")
    assert backend.endpoint_url == "http://garage"
    assert backend.access_key == "admin"
    assert backend.secret_key == "password"


def test_minio_init():
    """Test Minio backend initialization."""
    backend = Minio("http://minio", "admin", "password")
    assert backend.endpoint_url == "http://minio"
    assert backend.access_key == "admin"
    assert backend.secret_key == "password"
