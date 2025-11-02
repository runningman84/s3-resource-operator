import pytest

def test_prometheus_metrics():
    """Test that Prometheus metrics are being incremented."""
    # Import after the previous test to avoid conflicts
    from src.main import SECRETS_PROCESSED, ERRORS_TOTAL

    # Get initial values
    initial_secrets = SECRETS_PROCESSED._value.get()
    initial_errors = ERRORS_TOTAL._value.get()

    # Simulate processing a secret
    SECRETS_PROCESSED.inc()
    assert SECRETS_PROCESSED._value.get() == initial_secrets + 1

    # Simulate an error
    ERRORS_TOTAL.inc()
    assert ERRORS_TOTAL._value.get() == initial_errors + 1
