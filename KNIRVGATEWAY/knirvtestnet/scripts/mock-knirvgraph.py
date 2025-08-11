#!/usr/bin/env python3
"""
Mock KNIRVGRAPH server for testnet
Provides basic API endpoints for testing
"""

from http.server import HTTPServer, BaseHTTPRequestHandler
import json
import threading
import time
import sys

# Sample data storage
error_nodes = [
    {
        "id": "error_001",
        "description": "Division by zero in calculation module",
        "category": "arithmetic_error",
        "severity": "high",
        "context": {
            "module": "calculator",
            "function": "divide",
            "input": {"a": 10, "b": 0}
        }
    },
    {
        "id": "error_002",
        "description": "Memory allocation failure",
        "category": "memory_error", 
        "severity": "critical",
        "context": {
            "module": "allocator",
            "requested_size": "1GB",
            "available_memory": "512MB"
        }
    }
]

skill_nodes = [
    {
        "id": "skill_001",
        "name": "Safe Division",
        "description": "Performs division with zero-check",
        "resolves": ["error_001"],
        "code_hash": "QmSafeDivisionSkill...",
        "validation_proof": "proof_001"
    },
    {
        "id": "skill_002",
        "name": "Memory Manager",
        "description": "Safe memory allocation with fallback",
        "resolves": ["error_002"],
        "code_hash": "QmMemoryManagerSkill...",
        "validation_proof": "proof_002"
    }
]

class MockKNIRVGRAPHHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {
                "status": "healthy",
                "chain_id": "knirvgraph-testnet-1",
                "version": "testnet-1.0.0",
                "graph": {
                    "total_nodes": len(error_nodes) + len(skill_nodes),
                    "error_nodes": len(error_nodes),
                    "skill_nodes": len(skill_nodes)
                }
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/api/v1/error-nodes':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {"data": error_nodes}
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/api/v1/skill-nodes':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {"data": skill_nodes}
            self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        if self.path == '/api/v1/error-nodes':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            new_error = json.loads(post_data.decode())
            error_nodes.append(new_error)
            
            self.send_response(201)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {"result": "success", "id": new_error["id"]}
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/api/v1/skill-nodes':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            new_skill = json.loads(post_data.decode())
            skill_nodes.append(new_skill)
            
            self.send_response(201)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {"result": "success", "id": new_skill["id"]}
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/api/v1/ibc/skill-verification':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            response = {"result": "success", "verification_sent": True}
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
    server = HTTPServer(('0.0.0.0', 8081), MockKNIRVGRAPHHandler)
    print("Mock KNIRVGRAPH server started on http://localhost:8081")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nMock KNIRVGRAPH server stopped")
        server.shutdown()

if __name__ == '__main__':
    run_server()
