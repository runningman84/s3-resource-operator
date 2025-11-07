"""Tests for backend connection testing."""

import pytest
from unittest.mock import MagicMock, patch
from src.backends.versitygw import VersityGW
from src.backends.minio import Minio
from src.backends.garage import Garage


def test_versitygw_connection_success():
    """Test VersityGW connection test succeeds when backend is accessible."""
    backend = VersityGW(
        endpoint_url="http://test:8080",
        access_key="test-key",
        secret_key="test-secret"
    )

    # Mock the internal methods
    backend._list_buckets_raw = MagicMock(return_value=[
        {'Name': 'bucket1', 'Owner': 'user1'},
        {'Name': 'bucket2', 'Owner': 'user2'}
    ])
    backend._list_users_raw = MagicMock(return_value=[
        {'Access': 'user1'},
        {'Access': 'user2'}
    ])

    # Should not raise an exception
    backend.test_connection()

    # Verify both methods were called
    backend._list_buckets_raw.assert_called_once()
    backend._list_users_raw.assert_called_once()


def test_versitygw_connection_failure():
    """Test VersityGW connection test fails when backend is not accessible."""
    backend = VersityGW(
        endpoint_url="http://test:8080",
        access_key="test-key",
        secret_key="test-secret"
    )

    # Mock the internal method to raise an exception
    backend._list_buckets_raw = MagicMock(
        side_effect=Exception("Connection refused")
    )

    # Should raise an exception
    with pytest.raises(Exception) as exc_info:
        backend.test_connection()

    assert "Backend connection test failed" in str(exc_info.value)
    assert "Connection refused" in str(exc_info.value)


def test_minio_connection_not_implemented():
    """Test MinIO connection test raises NotImplementedError."""
    backend = Minio(
        endpoint_url="http://test:9000",
        access_key="test-key",
        secret_key="test-secret"
    )

    with pytest.raises(NotImplementedError):
        backend.test_connection()


def test_garage_connection_not_implemented():
    """Test Garage connection test raises NotImplementedError."""
    backend = Garage(
        endpoint_url="http://test:3900",
        access_key="test-key",
        secret_key="test-secret"
    )

    with pytest.raises(NotImplementedError):
        backend.test_connection()
