"""
Tests for the main KNIRVClient class.
"""

import pytest
from unittest.mock import patch, MagicMock

from knirv_sdk import KNIRVClient, ClientConfig
from knirv_sdk.exceptions import KNIRVSDKError


class TestKNIRVClient:
    """Test cases for KNIRVClient."""
    
    def test_client_initialization_default(self):
        """Test client initialization with default configuration."""
        client = KNIRVClient()
        
        assert client.config is not None
        assert client.config.environment == "production"
        assert not client.is_closed
    
    def test_client_initialization_custom_config(self, mock_config):
        """Test client initialization with custom configuration."""
        client = KNIRVClient(mock_config)
        
        assert client.config == mock_config
        assert client.config.environment == "testing"
        assert not client.is_closed
    
    def test_gateway_property(self, mock_config):
        """Test gateway service property access."""
        client = KNIRVClient(mock_config)
        
        gateway = client.gateway
        assert gateway is not None
        
        # Should return the same instance on subsequent calls
        assert client.gateway is gateway
    
    def test_transaction_property(self, mock_config):
        """Test transaction service property access."""
        client = KNIRVClient(mock_config)
        
        transaction = client.transaction
        assert transaction is not None
        
        # Should return the same instance on subsequent calls
        assert client.transaction is transaction
    
    def test_transmission_property(self, mock_config):
        """Test transmission service property access."""
        client = KNIRVClient(mock_config)
        
        transmission = client.transmission
        assert transmission is not None
        
        # Should return the same instance on subsequent calls
        assert client.transmission is transmission
    
    def test_closed_client_access(self, mock_config):
        """Test that accessing services on closed client raises error."""
        client = KNIRVClient(mock_config)
        client._closed = True
        
        with pytest.raises(KNIRVSDKError, match="Client has been closed"):
            _ = client.gateway
        
        with pytest.raises(KNIRVSDKError, match="Client has been closed"):
            _ = client.transaction
        
        with pytest.raises(KNIRVSDKError, match="Client has been closed"):
            _ = client.transmission
    
    @pytest.mark.asyncio
    async def test_async_context_manager(self, mock_config):
        """Test client as async context manager."""
        async with KNIRVClient(mock_config) as client:
            assert not client.is_closed
            gateway = client.gateway
            assert gateway is not None
        
        assert client.is_closed
    
    def test_sync_context_manager(self, mock_config):
        """Test client as sync context manager."""
        with KNIRVClient(mock_config) as client:
            assert not client.is_closed
            gateway = client.gateway
            assert gateway is not None
        
        assert client.is_closed
    
    @pytest.mark.asyncio
    async def test_close_method(self, mock_config):
        """Test explicit close method."""
        client = KNIRVClient(mock_config)
        
        # Access services to initialize them
        _ = client.gateway
        _ = client.transaction
        _ = client.transmission
        
        assert not client.is_closed
        
        await client.close()
        
        assert client.is_closed
    
    @pytest.mark.asyncio
    async def test_close_idempotent(self, mock_config):
        """Test that close can be called multiple times safely."""
        client = KNIRVClient(mock_config)
        
        await client.close()
        assert client.is_closed
        
        # Should not raise error
        await client.close()
        assert client.is_closed
    
    def test_repr(self, mock_config):
        """Test string representation of client."""
        client = KNIRVClient(mock_config)
        
        repr_str = repr(client)
        assert "KNIRVClient" in repr_str
        assert "testing" in repr_str
        assert "open" in repr_str
        
        client._closed = True
        repr_str = repr(client)
        assert "closed" in repr_str


class TestConvenienceFunctions:
    """Test convenience functions for creating clients."""
    
    def test_create_client(self):
        """Test create_client convenience function."""
        from knirv_sdk.client import create_client
        
        client = create_client(
            api_key="test-key",
            environment="development"
        )
        
        assert client.config.api_key == "test-key"
        assert client.config.environment == "development"
    
    def test_create_development_client(self):
        """Test create_development_client convenience function."""
        from knirv_sdk.client import create_development_client
        
        client = create_development_client()
        
        assert client.config.environment == "development"
        assert client.config.debug is True
        assert client.config.verbose is True
    
    def test_create_testing_client(self):
        """Test create_testing_client convenience function."""
        from knirv_sdk.client import create_testing_client
        
        client = create_testing_client()
        
        assert client.config.environment == "testing"
        assert client.config.debug is True
        assert client.config.timeout == 5
        assert client.config.retries == 1
