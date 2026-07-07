# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing_extensions import Literal, Annotated, TypedDict

from .._utils import PropertyInfo

__all__ = ["McpRetrieveContextsParams"]


class McpRetrieveContextsParams(TypedDict, total=False):
    capability_id: Annotated[str, PropertyInfo(alias="capabilityId")]
    """Filter by capability ID"""

    initiator: str
    """Filter by initiator address"""

    interaction_type: Annotated[
        Literal[
            "TOOL_INVOCATION",
            "PROMPT_USAGE",
            "RESOURCE_ACCESS",
            "PLUGIN_EXECUTION",
            "SAMPLING_REQUEST_SENT",
            "SAMPLING_RESPONSE_RECEIVED",
            "MEMORY_WRITE",
            "CAPABILITY_REGISTRATION",
        ],
        PropertyInfo(alias="interactionType"),
    ]
    """Filter by interaction type"""
