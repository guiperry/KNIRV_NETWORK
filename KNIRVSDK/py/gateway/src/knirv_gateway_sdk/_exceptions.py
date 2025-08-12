"""
Exception classes for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, Dict, Optional, Union
import httpx


class KNIRVGatewayError(Exception):
    """Base exception for all KNIRV Gateway SDK errors."""
    pass


class APIError(KNIRVGatewayError):
    """Base class for all API-related errors."""

    def __init__(
        self,
        message: str,
        *,
        request: Optional[httpx.Request] = None,
        response: Optional[httpx.Response] = None,
        status_code: Optional[int] = None,
    ) -> None:
        super().__init__(message)
        self.request = request
        self.response = response
        self.status_code = status_code


class APIConnectionError(APIError):
    """Error raised when unable to connect to the API."""
    
    def __init__(self, *, request: Optional[httpx.Request] = None) -> None:
        super().__init__("Connection error.", request=request)


class APITimeoutError(APIError):
    """Error raised when a request times out."""
    
    def __init__(self, *, request: Optional[httpx.Request] = None) -> None:
        super().__init__("Request timed out.", request=request)


class APIResponseValidationError(APIError):
    """Error raised when the API response cannot be validated."""
    
    def __init__(
        self,
        *,
        response: Optional[httpx.Response] = None,
        body: Optional[Union[str, bytes]] = None,
    ) -> None:
        super().__init__("Data returned by API invalid for expected schema.", response=response)
        self.body = body


class APIStatusError(APIError):
    """Error raised when the API returns an error status code."""
    
    def __init__(
        self,
        message: str,
        *,
        response: httpx.Response,
        body: Optional[Union[str, bytes, Dict[str, Any]]] = None,
    ) -> None:
        super().__init__(message, response=response)
        self.status_code = response.status_code
        self.body = body


class BadRequestError(APIStatusError):
    """Error raised when the API returns a 400 status code."""
    pass


class AuthenticationError(APIStatusError):
    """Error raised when the API returns a 401 status code."""
    pass


class PermissionDeniedError(APIStatusError):
    """Error raised when the API returns a 403 status code."""
    pass


class NotFoundError(APIStatusError):
    """Error raised when the API returns a 404 status code."""
    pass


class ConflictError(APIStatusError):
    """Error raised when the API returns a 409 status code."""
    pass


class UnprocessableEntityError(APIStatusError):
    """Error raised when the API returns a 422 status code."""
    pass


class RateLimitError(APIStatusError):
    """Error raised when the API returns a 429 status code."""
    pass


class InternalServerError(APIStatusError):
    """Error raised when the API returns a 500 status code."""
    pass


class ValidationError(KNIRVGatewayError):
    """Error raised when input validation fails."""
    pass


# Aliases for backward compatibility with tests
KNIRVAPIError = APIError
KNIRVValidationError = APIResponseValidationError
