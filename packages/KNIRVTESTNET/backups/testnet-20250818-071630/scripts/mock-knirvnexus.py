#!/usr/bin/env python3
"""
Mock KNIRV-NEXUS server for testnet
Provides basic API endpoints for testing
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import threading
import time
import sys

class MockKNIRVNEXUSHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/status':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "status": "healthy",
                "node_id": f"nexus-{sys.argv[1] if len(sys.argv) > 1 else '1'}",
                "version": "testnet-1.0.0",
                "tee": {
                    "simulation_mode": True,
                    "status": "ready"
                },
                "validation": {
                    "active_validations": 0,
                    "completed_validations": 10,
                    "success_rate": 0.95
                }
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/metrics':
            self.send_response(200)
            self.send_header('Content-type', 'text/plain')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            metrics = """# HELP nexus_validations_total Total number of validations
# TYPE nexus_validations_total counter
nexus_validations_total 10

# HELP nexus_validation_duration_seconds Validation duration
# TYPE nexus_validation_duration_seconds histogram
nexus_validation_duration_seconds_sum 5.0
nexus_validation_duration_seconds_count 10
"""
            self.wfile.write(metrics.encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == '/validate/skill':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            validation_request = json.loads(post_data.decode())
            
            # Simulate validation time
            time.sleep(0.1)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "result": "success",
                "validation_id": "val_001",
                "skill_code_valid": True,
                "test_results": {
                    "passed": 2,
                    "failed": 0,
                    "total": 2
                },
                "proof": "zkproof_abc123...",
                "execution_time": "0.1s"
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/validate/llm':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "result": "success",
                "validation_id": "val_002",
                "llm_valid": True,
                "performance_metrics": {
                    "accuracy": 0.95,
                    "latency": "50ms",
                    "throughput": "100 tokens/s"
                },
                "proof": "zkproof_def456..."
            }
            self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        self.end_headers()
    
    def log_message(self, format, *args):
        # Suppress default logging
        pass

def run_server():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8082
    server = HTTPServer(('0.0.0.0', port), MockKNIRVNEXUSHandler)
    print(f"Mock KNIRV-NEXUS server started on http://localhost:{port}")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print(f"\nMock KNIRV-NEXUS server stopped")
        server.shutdown()

if __name__ == '__main__':
    run_server()
