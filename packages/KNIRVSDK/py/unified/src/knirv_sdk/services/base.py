"""
Base Service Classes

This module provides base classes for all KNIRV SDK services.
"""

from __future__ import annotations
from typing import Optional, Dict, Any, Union
import asyncio
import httpx

from ..exceptions import (
    KNIRVAPIError,
    KNIRVConnectionError,
    KNIRVTimeoutError,
    KNIRVAuthenticationError,
    KNIRVNotFoundError,
    KNIRVRateLimitError,
    KNIRVServerError,
    KNIRVBadRequestError,
    KNIRVUnauthorizedError,
    KNIRVForbiddenError,
    KNIRVConflictError,
    KNIRVUnprocessableEntityError,
)


class BaseService:
    """
    Base class for all KNIRV services.
    
    Provides common functionality like HTTP client management,
    error handling, and configuration.
    """
    
    def __init__(self, base_url: str, config: Any) -> None:
        """
        Initialize the base service.

        Args:
            base_url: Base URL for the service
            config: Service configuration object
        """
        self._base_url = base_url
        self._config = config
        self.__http_client: Optional[httpx.AsyncClient] = None
        self._closed = False
    
    @property
    def _client(self) -> httpx.AsyncClient:
        """Get or create the HTTP client."""
        if self._closed:
            raise KNIRVAPIError("Service has been closed")

        if self.__http_client is None:
            # Build headers
            headers = {
                "User-Agent": self._config.user_agent,
                "Content-Type": "application/json",
                **self._config.default_headers,
            }
            
            if self._config.api_key:
                headers["Authorization"] = f"Bearer {self._config.api_key}"
            
            # Create HTTP client with retry configuration
            self.__http_client = httpx.AsyncClient(
                base_url=self._base_url,
                headers=headers,
                timeout=httpx.Timeout(self._config.timeout),
                limits=httpx.Limits(max_keepalive_connections=10, max_connections=100),
            )

        return self.__http_client
    
    @property
    def _http_client(self) -> httpx.AsyncClient:
        """Alias for _client property."""
        return self._client
    
    async def _request(
        self,
        method: str,
        url: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        json: Optional[Dict[str, Any]] = None,
        data: Optional[Union[str, bytes]] = None,
        headers: Optional[Dict[str, str]] = None,
        **kwargs: Any,
    ) -> httpx.Response:
        """
        Make an HTTP request with error handling and retries.
        
        Args:
            method: HTTP method (GET, POST, PUT, DELETE, etc.)
            url: Request URL (relative to base URL)
            params: Query parameters
            json: JSON data to send
            data: Raw data to send
            headers: Additional headers
            **kwargs: Additional arguments for httpx
        
        Returns:
            HTTP response
        
        Raises:
            KNIRVAPIError: For API errors
            KNIRVConnectionError: For connection errors
            KNIRVTimeoutError: For timeout errors
        """
        if self._closed:
            raise KNIRVAPIError("Service has been closed")
        
        # Merge headers
        request_headers = {}
        if headers:
            request_headers.update(headers)
        
        # Retry logic
        last_exception = None
        for attempt in range(self._config.retries + 1):
            try:
                response = await self._client.request(
                    method=method,
                    url=url,
                    params=params,
                    json=json,
                    data=data,
                    headers=request_headers,
                    **kwargs,
                )
                
                # Check for HTTP errors and convert to appropriate exceptions
                if response.status_code >= 400:
                    await self._handle_error_response(response)
                
                return response
                
            except httpx.TimeoutException as e:
                last_exception = KNIRVTimeoutError(f"Request timed out: {e}")
                if attempt == self._config.retries:
                    raise last_exception
                
            except httpx.ConnectError as e:
                last_exception = KNIRVConnectionError(f"Connection error: {e}")
                if attempt == self._config.retries:
                    raise last_exception
                
            except httpx.HTTPError as e:
                last_exception = KNIRVAPIError(f"HTTP error: {e}")
                if attempt == self._config.retries:
                    raise last_exception
            
            # Wait before retrying (exponential backoff)
            if attempt < self._config.retries:
                delay = min(
                    self._config.retry_delay * (2 ** attempt),
                    self._config.max_retry_delay
                )
                await asyncio.sleep(delay)
        
        # This should never be reached, but just in case
        if last_exception:
            raise last_exception
        raise KNIRVAPIError("Unexpected error in request retry logic")
    
    async def _handle_error_response(self, response: httpx.Response) -> None:
        """
        Handle error responses and raise appropriate exceptions.
        
        Args:
            response: HTTP response with error status
        
        Raises:
            Appropriate KNIRVAPIError subclass based on status code
        """
        try:
            error_data = response.json()
            message = error_data.get("message", f"HTTP {response.status_code}")
        except Exception:
            message = f"HTTP {response.status_code}: {response.text}"
        
        status_code = response.status_code
        
        if status_code == 400:
            raise KNIRVBadRequestError(message, status_code=status_code, response=error_data)
        elif status_code == 401:
            raise KNIRVUnauthorizedError(message, status_code=status_code, response=error_data)
        elif status_code == 403:
            raise KNIRVForbiddenError(message, status_code=status_code, response=error_data)
        elif status_code == 404:
            raise KNIRVNotFoundError(message, status_code=status_code, response=error_data)
        elif status_code == 409:
            raise KNIRVConflictError(message, status_code=status_code, response=error_data)
        elif status_code == 422:
            raise KNIRVUnprocessableEntityError(message, status_code=status_code, response=error_data)
        elif status_code == 429:
            raise KNIRVRateLimitError(message, status_code=status_code, response=error_data)
        elif status_code >= 500:
            raise KNIRVServerError(message, status_code=status_code, response=error_data)
        else:
            raise KNIRVAPIError(message, status_code=status_code, response=error_data)
    
    async def close(self) -> None:
        """Close the service and clean up resources."""
        if self._closed:
            return

        if self.__http_client is not None:
            await self.__http_client.aclose()
            self.__http_client = None

        self._closed = True
    
    async def __aenter__(self) -> BaseService:
        """Async context manager entry."""
        return self
    
    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Async context manager exit."""
        await self.close()
