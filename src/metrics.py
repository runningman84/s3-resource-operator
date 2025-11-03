"""Metrics server for exposing Prometheus metrics and health checks."""

import logging
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer
from prometheus_client import generate_latest

logger = logging.getLogger("s3-resource-operator")


class MetricsServer:
    """HTTP server for exposing Prometheus metrics and health endpoints."""

    def __init__(self, port=8000):
        self.port = port
        self.server = HTTPServer(('', self.port), self._get_handler())

    def _get_handler(self):
        class MetricsHandler(BaseHTTPRequestHandler):
            def do_GET(self):
                if self.path == '/metrics':
                    self.send_response(200)
                    self.send_header('Content-type', 'text/plain')
                    self.end_headers()
                    self.wfile.write(generate_latest())
                elif self.path == '/healthz':
                    self.send_response(200)
                    self.send_header('Content-type', 'application/json')
                    self.end_headers()
                    self.wfile.write(b'{"status":"healthy"}')
                else:
                    self.send_response(404)
                    self.end_headers()
                    self.wfile.write(b"Not Found")
        return MetricsHandler

    def start(self):
        """Start the metrics server in a background thread."""
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.daemon = True
        self.thread.start()
        logger.info(f"Metrics server started on port {self.port} at /metrics")
