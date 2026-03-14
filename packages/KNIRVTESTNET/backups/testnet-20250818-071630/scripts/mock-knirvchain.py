#!/usr/bin/env python3
"""
Mock KNIRVCHAIN server for testnet
Provides basic API endpoints for testing
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import threading
import time
import sys

class MockKNIRVCHAINHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "status": "healthy",
                "chain_id": "knirvchain-testnet-1",
                "version": "testnet-1.0.0",
                "base_llm": {
                    "model": "CodeT5",
                    "version": "1.0.0",
                    "status": "ready"
                },
                "skill_registry": {
                    "total_skills": 3,
                    "validated_skills": 3
                }
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/v2/skills':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "skills": [
                    {
                        "id": "skill_001",
                        "name": "Safe Division",
                        "description": "Performs division with zero-check",
                        "status": "validated"
                    },
                    {
                        "id": "skill_002", 
                        "name": "Memory Manager",
                        "description": "Safe memory allocation with fallback",
                        "status": "validated"
                    },
                    {
                        "id": "skill_003",
                        "name": "Safe JSON Parser", 
                        "description": "JSON parser with error handling",
                        "status": "validated"
                    }
                ]
            }
            self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == '/v2/skill/invoke':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "result": "success",
                "skill_id": "skill_001",
                "execution_time": "0.1s",
                "nrn_burned": 10,
                "output": {"result": True}
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/v2/skill/register':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "result": "success",
                "skill_id": "skill_new",
                "status": "registered",
                "validation_required": True
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
    server = HTTPServer(('0.0.0.0', 8080), MockKNIRVCHAINHandler)
    print("Mock KNIRVCHAIN server started on http://localhost:8080")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nMock KNIRVCHAIN server stopped")
        server.shutdown()

if __name__ == '__main__':
    run_server()
