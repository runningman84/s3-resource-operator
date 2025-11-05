"""Tests for Prometheus metrics."""

import pytest
from src.operator import (
    SECRETS_PROCESSED,
    ERRORS_TOTAL,
    USERS_CREATED,
    USERS_DELETED,
    USERS_UPDATED,
    BUCKETS_CREATED,
    BUCKET_OWNERS_CHANGED
)


def test_prometheus_metrics():
    """Test that Prometheus metrics are being incremented."""
    # Get initial values
    initial_secrets = SECRETS_PROCESSED._value.get()
    initial_errors = ERRORS_TOTAL._value.get()

    # Simulate processing a secret
    SECRETS_PROCESSED.inc()
    assert SECRETS_PROCESSED._value.get() == initial_secrets + 1

    # Simulate an error
    ERRORS_TOTAL.inc()
    assert ERRORS_TOTAL._value.get() == initial_errors + 1


def test_resource_operation_metrics():
    """Test that resource operation metrics are being incremented."""
    # Get initial values
    initial_users_created = USERS_CREATED._value.get()
    initial_users_deleted = USERS_DELETED._value.get()
    initial_users_updated = USERS_UPDATED._value.get()
    initial_buckets_created = BUCKETS_CREATED._value.get()
    initial_owners_changed = BUCKET_OWNERS_CHANGED._value.get()

    # Simulate operations
    USERS_CREATED.inc()
    assert USERS_CREATED._value.get() == initial_users_created + 1

    USERS_DELETED.inc()
    assert USERS_DELETED._value.get() == initial_users_deleted + 1

    USERS_UPDATED.inc()
    assert USERS_UPDATED._value.get() == initial_users_updated + 1

    BUCKETS_CREATED.inc()
    assert BUCKETS_CREATED._value.get() == initial_buckets_created + 1

    BUCKET_OWNERS_CHANGED.inc()
    assert BUCKET_OWNERS_CHANGED._value.get() == initial_owners_changed + 1
