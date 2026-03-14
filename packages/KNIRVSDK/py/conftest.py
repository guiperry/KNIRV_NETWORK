"""
Pytest configuration for KNIRV Python SDK
"""
import pytest
import sys
import os

# Add the source directories to Python path
current_dir = os.path.dirname(os.path.abspath(__file__))
gateway_src = os.path.join(current_dir, 'gateway', 'src')
transaction_src = os.path.join(current_dir, 'transaction', 'src')
transmission_src = os.path.join(current_dir, 'transmission')

for src_path in [gateway_src, transaction_src, transmission_src]:
    if os.path.exists(src_path) and src_path not in sys.path:
        sys.path.insert(0, src_path)

@pytest.fixture(scope="session")
def test_config():
    """Provide test configuration"""
    return {
        "base_url": "http://localhost:8000",
        "timeout": 30,
        "retries": 3
    }

@pytest.fixture
def mock_response():
    """Provide a mock response for testing"""
    class MockResponse:
        def __init__(self, json_data, status_code=200):
            self.json_data = json_data
            self.status_code = status_code
            
        def json(self):
            return self.json_data
            
        def raise_for_status(self):
            if self.status_code >= 400:
                raise Exception(f"HTTP {self.status_code}")
                
    return MockResponse
