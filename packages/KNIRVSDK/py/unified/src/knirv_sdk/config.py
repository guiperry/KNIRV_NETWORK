"""
KNIRV SDK Configuration Management

This module provides configuration classes for the unified KNIRV SDK.
"""

from __future__ import annotations
import os
from typing import Optional, Dict, Any
from dataclasses import dataclass, field


@dataclass
class GatewayConfig:
    """Configuration for Gateway service."""
    
    base_url: str = "https://gateway.knirv.com"
    api_key: Optional[str] = None
    timeout: int = 30
    retries: int = 3
    retry_delay: int = 1
    max_retry_delay: int = 10
    user_agent: str = "knirv-sdk-python/1.0.0"
    default_headers: Dict[str, str] = field(default_factory=dict)
    
    def __post_init__(self) -> None:
        # Load from environment variables if not provided
        if self.api_key is None:
            self.api_key = os.getenv("KNIRV_GATEWAY_API_KEY")
        
        if self.base_url == "https://gateway.knirv.com":
            env_url = os.getenv("KNIRV_GATEWAY_URL")
            if env_url:
                self.base_url = env_url


@dataclass
class TransactionConfig:
    """Configuration for Transaction service."""
    
    base_url: str = "https://transaction.knirv.com"
    api_key: Optional[str] = None
    timeout: int = 30
    retries: int = 3
    retry_delay: int = 1
    max_retry_delay: int = 10
    user_agent: str = "knirv-sdk-python/1.0.0"
    default_headers: Dict[str, str] = field(default_factory=dict)
    
    def __post_init__(self) -> None:
        # Load from environment variables if not provided
        if self.api_key is None:
            self.api_key = os.getenv("KNIRV_TRANSACTION_API_KEY")
        
        if self.base_url == "https://transaction.knirv.com":
            env_url = os.getenv("KNIRV_TRANSACTION_URL")
            if env_url:
                self.base_url = env_url


@dataclass
class TransmissionConfig:
    """Configuration for Transmission service."""
    
    base_url: str = "https://transmission.knirv.com"
    api_key: Optional[str] = None
    timeout: int = 30
    retries: int = 3
    retry_delay: int = 1
    max_retry_delay: int = 10
    user_agent: str = "knirv-sdk-python/1.0.0"
    default_headers: Dict[str, str] = field(default_factory=dict)
    
    def __post_init__(self) -> None:
        # Load from environment variables if not provided
        if self.api_key is None:
            self.api_key = os.getenv("KNIRV_TRANSMISSION_API_KEY")
        
        if self.base_url == "https://transmission.knirv.com":
            env_url = os.getenv("KNIRV_TRANSMISSION_URL")
            if env_url:
                self.base_url = env_url


@dataclass
class ClientConfig:
    """Unified configuration for all KNIRV services."""
    
    # Global settings
    api_key: Optional[str] = None
    timeout: int = 30
    retries: int = 3
    retry_delay: int = 1
    max_retry_delay: int = 10
    user_agent: str = "knirv-sdk-python/1.0.0"
    default_headers: Dict[str, str] = field(default_factory=dict)
    
    # Service-specific configurations
    gateway: Optional[GatewayConfig] = None
    transaction: Optional[TransactionConfig] = None
    transmission: Optional[TransmissionConfig] = None
    
    # Environment settings
    environment: str = "production"  # production, staging, development
    debug: bool = False
    verbose: bool = False
    
    def __post_init__(self) -> None:
        # Load global API key from environment if not provided
        if self.api_key is None:
            self.api_key = os.getenv("KNIRV_API_KEY")
        
        # Load environment setting
        env_environment = os.getenv("KNIRV_ENVIRONMENT")
        if env_environment:
            self.environment = env_environment
        
        # Load debug setting
        env_debug = os.getenv("KNIRV_DEBUG")
        if env_debug:
            self.debug = env_debug.lower() in ("true", "1", "yes", "on")
        
        # Load verbose setting
        env_verbose = os.getenv("KNIRV_VERBOSE")
        if env_verbose:
            self.verbose = env_verbose.lower() in ("true", "1", "yes", "on")
        
        # Initialize service configs if not provided
        if self.gateway is None:
            self.gateway = GatewayConfig(
                api_key=self.api_key,
                timeout=self.timeout,
                retries=self.retries,
                retry_delay=self.retry_delay,
                max_retry_delay=self.max_retry_delay,
                user_agent=self.user_agent,
                default_headers=self.default_headers.copy()
            )
        
        if self.transaction is None:
            self.transaction = TransactionConfig(
                api_key=self.api_key,
                timeout=self.timeout,
                retries=self.retries,
                retry_delay=self.retry_delay,
                max_retry_delay=self.max_retry_delay,
                user_agent=self.user_agent,
                default_headers=self.default_headers.copy()
            )
        
        if self.transmission is None:
            self.transmission = TransmissionConfig(
                api_key=self.api_key,
                timeout=self.timeout,
                retries=self.retries,
                retry_delay=self.retry_delay,
                max_retry_delay=self.max_retry_delay,
                user_agent=self.user_agent,
                default_headers=self.default_headers.copy()
            )
    
    @classmethod
    def from_environment(cls) -> ClientConfig:
        """Create configuration from environment variables."""
        return cls()
    
    @classmethod
    def for_development(cls, **kwargs: Any) -> ClientConfig:
        """Create configuration for development environment."""
        config = cls(
            environment="development",
            debug=True,
            verbose=True,
            **kwargs
        )
        
        # Use development URLs if available
        if config.gateway:
            config.gateway.base_url = os.getenv("KNIRV_GATEWAY_DEV_URL", config.gateway.base_url)
        if config.transaction:
            config.transaction.base_url = os.getenv("KNIRV_TRANSACTION_DEV_URL", config.transaction.base_url)
        if config.transmission:
            config.transmission.base_url = os.getenv("KNIRV_TRANSMISSION_DEV_URL", config.transmission.base_url)
        
        return config
    
    @classmethod
    def for_testing(cls, **kwargs: Any) -> ClientConfig:
        """Create configuration for testing environment."""
        return cls(
            environment="testing",
            debug=True,
            verbose=False,
            timeout=5,  # Shorter timeout for tests
            retries=1,  # Fewer retries for tests
            **kwargs
        )
