#!/usr/bin/env python3
"""
Mock KNIRV-ROOT server for testnet
Provides basic API endpoints for testing
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import threading
import time
import sys

class MockKNIRVROOTHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/status':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "node_info": {
                    "network": "knirv-testnet-1",
                    "version": "testnet-1.0.0",
                    "moniker": "knirv-testnet-root"
                },
                "sync_info": {
                    "latest_block_height": "100",
                    "latest_block_time": "2025-08-06T15:00:00Z",
                    "catching_up": False
                },
                "validator_info": {
                    "address": "knirv1validator1...",
                    "voting_power": "100"
                }
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path.startswith('/bank/balances/'):
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "balances": [
                    {"denom": "unrn", "amount": "1000000"}
                ],
                "pagination": {"next_key": None, "total": "1"}
            }
            self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == '/ibc/transfer':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "tx_hash": "ABC123...",
                "code": 0,
                "raw_log": "transfer successful"
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
    server = HTTPServer(('0.0.0.0', 1317), MockKNIRVROOTHandler)
    print("Mock KNIRV-ROOT server started on http://localhost:1317")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nMock KNIRV-ROOT server stopped")
        server.shutdown()

if __name__ == '__main__':
    run_server()
