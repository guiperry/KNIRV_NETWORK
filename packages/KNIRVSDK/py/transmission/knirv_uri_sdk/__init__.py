"""
KNIRV Client SDK for Python.

This package provides a high-level interface for interacting with the KNIRVCHAIN network.
It simplifies the process of resolving `knirv://` URIs, discovering peers on the private DHT,
connecting to them, and fetching the underlying resources.
"""

from .parser import KnirvURI, parse_knirv_uri, KnirvURIError
from .client import KnirvClient, KnirvClientError, ResourceData

__all__ = [
    'KnirvURI',
    'parse_knirv_uri',
    'KnirvURIError',
    'KnirvClient',
    'KnirvClientError',
    'ResourceData',
]

__version__ = '0.1.0'