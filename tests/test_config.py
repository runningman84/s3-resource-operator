"""Tests for configuration and environment variable loading."""

import pytest
from unittest.mock import patch, MagicMock
import os
import sys


def test_load_dotenv():
    """Test loading of environment variables from a .env file."""
    os.environ['TEST_MODE'] = 'true'

    # Remove the module from sys.modules first
    if 'src.main' in sys.modules:
        del sys.modules['src.main']

    # Now patch before importing
    with patch('dotenv.load_dotenv') as mock_load_dotenv, \
            patch('src.main.MetricsServer') as mock_metrics_server, \
            patch('src.main.get_k8s_api') as mock_get_k8s_api, \
            patch('src.main.get_backend') as mock_get_backend, \
            patch('src.main.SecretManager') as mock_secret_manager, \
            patch('src.main.Operator') as mock_operator:

        # Set up required environment variables
        os.environ['S3_ENDPOINT_URL'] = 'http://test:9000'
        os.environ['S3_ACCESS_KEY'] = 'test-key'
        os.environ['S3_SECRET_KEY'] = 'test-secret'
        os.environ['BACKEND_NAME'] = 'versitygw'

        # Mock the operator instance
        mock_operator_instance = MagicMock()
        mock_operator.return_value = mock_operator_instance

        # Import and run - expect SystemExit(0) for clean shutdown
        import src.main
        with pytest.raises(SystemExit) as exc_info:
            src.main.main()

        # Verify clean exit
        assert exc_info.value.code == 0

        # Verify load_dotenv was called
        mock_load_dotenv.assert_called_once()

        # Clean up environment variables
        del os.environ['S3_ENDPOINT_URL']
        del os.environ['S3_ACCESS_KEY']
        del os.environ['S3_SECRET_KEY']
        del os.environ['BACKEND_NAME']

    del os.environ['TEST_MODE']
