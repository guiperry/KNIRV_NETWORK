#!/usr/bin/env python3
"""
Mock KNIRV-GATEWAY server for testnet
Provides unified API gateway for all services
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import urllib.request
import urllib.parse
import threading
import time
import sys

class MockKNIRVGATEWAYHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "status": "healthy",
                "version": "testnet-1.0.0",
                "services": {
                    "knirvoracle": "http://localhost:1317",
                    "knirvchain": "http://localhost:8080",
                    "knirvgraph": "http://localhost:8081",
                    "knirvnexus": ["http://localhost:8082", "http://localhost:8083"],
                    "knirvrouter": "http://localhost:8086"
                },
                "load_balancer": "round_robin",
                "authentication": True,
                "rate_limiting": True
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/metrics':
            self.send_response(200)
            self.send_header('Content-type', 'text/plain')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            metrics = """# HELP gateway_requests_total Total number of requests
# TYPE gateway_requests_total counter
gateway_requests_total 100

# HELP gateway_request_duration_seconds Request duration
# TYPE gateway_request_duration_seconds histogram
gateway_request_duration_seconds_sum 10.0
gateway_request_duration_seconds_count 100
"""
            self.wfile.write(metrics.encode())
        
        # Proxy requests to backend services
        elif self.path.startswith('/api/root/'):
            self.proxy_request('http://localhost:1317', self.path.replace('/api/root', ''))
        elif self.path.startswith('/api/chain/'):
            self.proxy_request('http://localhost:8080', self.path.replace('/api/chain', ''))
        elif self.path.startswith('/api/graph/'):
            self.proxy_request('http://localhost:8081', self.path.replace('/api/graph', ''))
        elif self.path.startswith('/api/nexus/'):
            self.proxy_request('http://localhost:8082', self.path.replace('/api/nexus', ''))
        elif self.path.startswith('/api/router/'):
            self.proxy_request('http://localhost:8086', self.path.replace('/api/router', ''))
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        # Proxy POST requests to backend services
        if self.path.startswith('/api/root/'):
            self.proxy_request('http://localhost:1317', self.path.replace('/api/root', ''), method='POST')
        elif self.path.startswith('/api/chain/'):
            self.proxy_request('http://localhost:8080', self.path.replace('/api/chain', ''), method='POST')
        elif self.path.startswith('/api/graph/'):
            self.proxy_request('http://localhost:8081', self.path.replace('/api/graph', ''), method='POST')
        elif self.path.startswith('/api/nexus/'):
            self.proxy_request('http://localhost:8082', self.path.replace('/api/nexus', ''), method='POST')
        elif self.path.startswith('/api/router/'):
            self.proxy_request('http://localhost:8086', self.path.replace('/api/router', ''), method='POST')
        else:
            self.send_response(404)
            self.end_headers()
    
    def proxy_request(self, backend_url, path, method='GET'):
        try:
            url = f"{backend_url}{path}"
            
            if method == 'POST':
                content_length = int(self.headers.get('Content-Length', 0))
                post_data = self.rfile.read(content_length) if content_length > 0 else None
                
                req = urllib.request.Request(url, data=post_data, method='POST')
                if post_data:
                    req.add_header('Content-Type', 'application/json')
            else:
                req = urllib.request.Request(url, method='GET')
            
            with urllib.request.urlopen(req, timeout=5) as response:
                self.send_response(response.getcode())
                self.send_header('Content-type', response.headers.get('Content-Type', 'application/json'))
                self.send_header('Access-Control-Allow-Origin', '*')
                self.end_headers()
                self.wfile.write(response.read())
        
        except Exception as e:
            # Return mock response if backend is not available
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            mock_response = {
                "status": "mock_response",
                "message": f"Backend service unavailable, returning mock data",
                "path": path,
                "backend": backend_url
            }
            self.wfile.write(json.dumps(mock_response).encode())
    
    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type, Authorization')
        self.end_headers()
    
    def log_message(self, format, *args):
        # Suppress default logging
        pass

def run_server():
    server = HTTPServer(('0.0.0.0', 8087), MockKNIRVGATEWAYHandler)
    print("Mock KNIRV-GATEWAY server started on http://localhost:8087")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nMock KNIRV-GATEWAY server stopped")
        server.shutdown()

if __name__ == '__main__':
    run_server()
