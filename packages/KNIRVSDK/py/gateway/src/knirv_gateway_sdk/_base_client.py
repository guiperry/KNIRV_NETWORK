"""
Base client implementation for the KNIRV Gateway SDK
"""

from __future__ import annotations

import json
import logging
from typing import Any, Dict, Optional, Union, cast
import httpx

from ._exceptions import (
    APIConnectionError,
    APITimeoutError,
    APIStatusError,
    BadRequestError,
    AuthenticationError,
    PermissionDeniedError,
    NotFoundError,
    ConflictError,
    UnprocessableEntityError,
    RateLimitError,
    InternalServerError,
)

logger = logging.getLogger(__name__)


class BaseClient:
    """Base client for making HTTP requests to the KNIRV Gateway API."""
    
    def __init__(
        self,
        *,
        base_url: str,
        timeout: Union[float, httpx.Timeout] = 60.0,
        max_retries: int = 2,
        default_headers: Optional[Dict[str, str]] = None,
        default_query: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> None:
        """
        Initialize the base client.
        
        Args:
            base_url: Base URL for the API
            timeout: Request timeout
            max_retries: Maximum number of retries
            default_headers: Default headers for all requests
            default_query: Default query parameters for all requests
            **kwargs: Additional arguments
        """
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.max_retries = max_retries
        self._default_headers = default_headers or {}
        self._default_query = default_query or {}
        
        # Create HTTP client
        self._client = httpx.AsyncClient(
            base_url=self.base_url,
            timeout=timeout,
            **kwargs,
        )
    
    async def __aenter__(self) -> "BaseClient":
        """Async context manager entry."""
        return self
    
    async def __aexit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        """Async context manager exit."""
        await self.close()
    
    async def close(self) -> None:
        """Close the HTTP client."""
        await self._client.aclose()
    
    def _build_headers(self, headers: Optional[Dict[str, str]] = None) -> Dict[str, str]:
        """Build headers for a request."""
        request_headers = {
            "User-Agent": "knirv-gateway-sdk-python/0.0.1",
            "Content-Type": "application/json",
            **self._default_headers,
        }
        
        # Add auth headers if available
        if hasattr(self, "auth_headers"):
            request_headers.update(self.auth_headers)
        
        if headers:
            request_headers.update(headers)
        
        return request_headers
    
    def _build_params(self, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Build query parameters for a request."""
        request_params = {**self._default_query}
        if params:
            request_params.update(params)
        return request_params
    
    def _handle_error_response(self, response: httpx.Response) -> None:
        """Handle error responses from the API."""
        try:
            error_data = response.json()
            message = error_data.get("message", f"HTTP {response.status_code}")
        except (json.JSONDecodeError, ValueError):
            message = f"HTTP {response.status_code}"
        
        if response.status_code == 400:
            raise BadRequestError(message, response=response, body=response.content)
        elif response.status_code == 401:
            raise AuthenticationError(message, response=response, body=response.content)
        elif response.status_code == 403:
            raise PermissionDeniedError(message, response=response, body=response.content)
        elif response.status_code == 404:
            raise NotFoundError(message, response=response, body=response.content)
        elif response.status_code == 409:
            raise ConflictError(message, response=response, body=response.content)
        elif response.status_code == 422:
            raise UnprocessableEntityError(message, response=response, body=response.content)
        elif response.status_code == 429:
            raise RateLimitError(message, response=response, body=response.content)
        elif response.status_code >= 500:
            raise InternalServerError(message, response=response, body=response.content)
        else:
            raise APIStatusError(message, response=response, body=response.content)
    
    async def request(
        self,
        method: str,
        path: str,
        *,
        params: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        json_data: Optional[Dict[str, Any]] = None,
        **kwargs: Any,
    ) -> httpx.Response:
        """
        Make an HTTP request to the API.
        
        Args:
            method: HTTP method
            path: API path
            params: Query parameters
            headers: Request headers
            json_data: JSON data for the request body
            **kwargs: Additional arguments for the request
            
        Returns:
            HTTP response
            
        Raises:
            APIConnectionError: If unable to connect to the API
            APITimeoutError: If the request times out
            APIStatusError: If the API returns an error status code
        """
        url = f"{self.base_url}/{path.lstrip('/')}"
        request_headers = self._build_headers(headers)
        request_params = self._build_params(params)
        
        try:
            response = await self._client.request(
                method=method,
                url=url,
                headers=request_headers,
                params=request_params,
                json=json_data,
                **kwargs,
            )
            
            if not response.is_success:
                self._handle_error_response(response)
            
            return response
            
        except httpx.ConnectError as e:
            raise APIConnectionError(request=e.request) from e
        except httpx.TimeoutException as e:
            raise APITimeoutError(request=e.request) from e
    
    async def get(self, path: str, **kwargs: Any) -> httpx.Response:
        """Make a GET request."""
        return await self.request("GET", path, **kwargs)
    
    async def post(self, path: str, **kwargs: Any) -> httpx.Response:
        """Make a POST request."""
        return await self.request("POST", path, **kwargs)
    
    async def put(self, path: str, **kwargs: Any) -> httpx.Response:
        """Make a PUT request."""
        return await self.request("PUT", path, **kwargs)
    
    async def patch(self, path: str, **kwargs: Any) -> httpx.Response:
        """Make a PATCH request."""
        return await self.request("PATCH", path, **kwargs)
    
    async def delete(self, path: str, **kwargs: Any) -> httpx.Response:
        """Make a DELETE request."""
        return await self.request("DELETE", path, **kwargs)
