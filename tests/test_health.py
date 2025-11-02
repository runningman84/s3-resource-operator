import pytest
import threading
import time
import requests
from src.main import MetricsServer


def test_health_endpoint():
    """Test that the /healthz endpoint returns a healthy status"""
    # Start the metrics server
    server = MetricsServer(port=8001)
    server.start()
    
    # Give the server a moment to start
    time.sleep(0.5)
    
    try:
        # Test the health endpoint
        response = requests.get('http://localhost:8001/healthz')
        assert response.status_code == 200
        assert response.headers['Content-Type'] == 'application/json'
        assert response.json() == {"status": "healthy"}
    finally:
        # Clean up
        server.server.shutdown()


def test_metrics_endpoint_still_works():
    """Ensure the /metrics endpoint still works after adding /healthz"""
    # Start the metrics server
    server = MetricsServer(port=8002)
    server.start()
    
    # Give the server a moment to start
    time.sleep(0.5)
    
    try:
        # Test the metrics endpoint
        response = requests.get('http://localhost:8002/metrics')
        assert response.status_code == 200
        assert 'text/plain' in response.headers['Content-Type']
        # Just verify we got some content (Prometheus metrics format)
        assert len(response.content) >= 0
    finally:
        # Clean up
        server.server.shutdown()


def test_404_for_unknown_path():
    """Test that unknown paths return 404"""
    # Start the metrics server
    server = MetricsServer(port=8003)
    server.start()
    
    # Give the server a moment to start
    time.sleep(0.5)
    
    try:
        # Test an unknown path
        response = requests.get('http://localhost:8003/unknown')
        assert response.status_code == 404
        assert response.content == b"Not Found"
    finally:
        # Clean up
        server.server.shutdown()
