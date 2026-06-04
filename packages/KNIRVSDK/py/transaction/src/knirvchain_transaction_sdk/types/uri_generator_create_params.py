# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Dict
from typing_extensions import TypedDict

__all__ = ["UriGeneratorCreateParams"]


class UriGeneratorCreateParams(TypedDict, total=False):
    content_hash: str
    """Hash of the content"""

    metadata: Dict[str, object]
    """Additional metadata"""

    owner: str
    """Owner address"""

    resource_type: str
    """Type of resource"""
