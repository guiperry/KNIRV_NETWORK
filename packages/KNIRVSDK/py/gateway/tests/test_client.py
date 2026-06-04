"""
Tests for the KNIRV Gateway client
"""

import pytest
from knirv_gateway_sdk import KNIRVGateway


def test_client_initialization():
    """Test client initialization."""
    client = KNIRVGateway(
        base_url="https://test.knirv.com",
        api_key="test-key"
    )
    
    assert client.base_url == "https://test.knirv.com"
    assert client._api_key == "test-key"
    
    # Check that all services are initialized
    assert client.economics is not None
    assert client.gateway is not None
    assert client.health is not None
    assert client.integration is not None
    assert client.poaud is not None


def test_economics_client_creation():
    """Test economics client creation."""
    client = KNIRVGateway.create_economics_client(
        base_url="https://economics.knirv.com",
        api_key="test-key"
    )
    
    assert client.base_url == "https://economics.knirv.com"
    assert client._api_key == "test-key"


def test_gateway_client_creation():
    """Test gateway client creation."""
    client = KNIRVGateway.create_gateway_client(
        base_url="https://gateway.knirv.com",
        api_key="test-key"
    )
    
    assert client.base_url == "https://gateway.knirv.com"
    assert client._api_key == "test-key"


def test_auth_headers():
    """Test authentication headers."""
    client = KNIRVGateway(
        base_url="https://test.knirv.com",
        api_key="test-key"
    )
    
    headers = client.auth_headers
    assert headers["Authorization"] == "Bearer test-key"


def test_auth_headers_no_key():
    """Test authentication headers without API key."""
    client = KNIRVGateway(
        base_url="https://test.knirv.com"
    )
    
    headers = client.auth_headers
    assert "Authorization" not in headers
