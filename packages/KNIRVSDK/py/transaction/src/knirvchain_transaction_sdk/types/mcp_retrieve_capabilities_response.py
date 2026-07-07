# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import List
from typing_extensions import TypeAlias

from .mcp.capability_descriptor import CapabilityDescriptor

__all__ = ["McpRetrieveCapabilitiesResponse"]

McpRetrieveCapabilitiesResponse: TypeAlias = List[CapabilityDescriptor]
