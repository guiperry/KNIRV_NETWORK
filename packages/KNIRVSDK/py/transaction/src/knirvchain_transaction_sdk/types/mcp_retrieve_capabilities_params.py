# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing_extensions import Literal, TypedDict

__all__ = ["McpRetrieveCapabilitiesParams"]


class McpRetrieveCapabilitiesParams(TypedDict, total=False):
    owner: str
    """Filter by owner address"""

    type: Literal["RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE"]
    """Filter by capability type"""
