#!/usr/bin/env python3
"""
Mock KNIRV-ROUTER server for testnet
Provides basic API endpoints for testing
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import threading
import time
import sys

class MockKNIRVROUTERHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/status':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "status": "healthy",
                "node_id": "router-testnet-1",
                "version": "testnet-1.0.0",
                "connectivity": {
                    "active_connections": 5,
                    "total_proofs_generated": 25,
                    "nrn_minted": 2500
                },
                "turn_server": {
                    "enabled": True,
                    "port": 3478,
                    "active_sessions": 2
                }
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/peers':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "peers": [
                    {"id": "peer1", "address": "192.168.1.100", "status": "connected"},
                    {"id": "peer2", "address": "192.168.1.101", "status": "connected"},
                    {"id": "peer3", "address": "192.168.1.102", "status": "connected"}
                ]
            }
            self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == '/connectivity/proof':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            proof_request = json.loads(post_data.decode())
            
            # Simulate connectivity proof generation
            time.sleep(0.2)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "result": "success",
                "proof_id": "proof_001",
                "paths_tested": len(proof_request.get("paths", [])),
                "successful_paths": len(proof_request.get("paths", [])),
                "nrn_minted": 100,
                "proof_hash": "0xabc123...",
                "timestamp": "2025-08-06T15:00:00Z"
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/nrn/mint':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "result": "success",
                "amount_minted": 100,
                "tx_hash": "0xdef456...",
                "recipient": "knirv1router..."
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
    server = HTTPServer(('0.0.0.0', 8086), MockKNIRVROUTERHandler)
    print("Mock KNIRV-ROUTER server started on http://localhost:8086")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nMock KNIRV-ROUTER server stopped")
        server.shutdown()

if __name__ == '__main__':
    run_server()
