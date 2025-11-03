"""Tests for shutdown and signal handling."""

import pytest
from unittest.mock import MagicMock, patch
import signal
import threading
import time
from src.operator import Operator


def test_signal_handling():
    """Test that the operator handles SIGTERM and SIGINT gracefully."""

    # Create a mock operator
    v1_api = MagicMock()
    backend = MagicMock()
    secret_manager = MagicMock()

    operator = Operator(v1_api, backend, secret_manager)

    # Verify shutdown event is not set initially
    assert not operator._shutdown.is_set()

    # Call shutdown
    operator.shutdown()

    # Verify shutdown event is now set
    assert operator._shutdown.is_set()


def test_graceful_watch_shutdown():
    """Test that the watch loop respects the shutdown signal."""
    from src.main import Operator

    # Create mocks
    v1_api = MagicMock()
    backend = MagicMock()
    secret_manager = MagicMock()
    secret_manager.annotation_key = "test-key"

    operator = Operator(v1_api, backend, secret_manager)

    # Mock the watch stream to yield events indefinitely
    mock_watch = MagicMock()
    mock_event = {
        'object': MagicMock(),
        'type': 'ADDED'
    }
    mock_event['object'].metadata.annotations = {}

    # Make stream generator that checks shutdown
    def stream_generator(*args, **kwargs):
        while not operator._shutdown.is_set():
            yield mock_event

    mock_watch.stream.return_value = stream_generator()

    with patch('src.operator.watch.Watch', return_value=mock_watch):
        # Start watch in a thread
        watch_thread = threading.Thread(target=operator.watch)
        watch_thread.start()

        # Give it a moment to start
        time.sleep(0.1)

        # Signal shutdown
        operator.shutdown()

        # Wait for watch to exit
        watch_thread.join(timeout=2)

        # Verify thread exited
        assert not watch_thread.is_alive()

        # Verify watch.stop() was called
        mock_watch.stop.assert_called_once()
