# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Dict, List, Union
from typing_extensions import Literal, Annotated, TypeAlias, TypedDict

from ..._utils import PropertyInfo
from .base_descriptor_param import BaseDescriptorParam

__all__ = [
    "CapabilityDescriptorParam",
    "ResourceDescriptor",
    "ResourceDescriptorSchema",
    "ToolDescriptor",
    "PromptDescriptor",
    "MemoryServiceDescriptor",
]


class ResourceDescriptorSchema(TypedDict, total=False):
    access_info: Annotated[Dict[str, object], PropertyInfo(alias="accessInfo")]
    """For APIs - endpoint, auth details, etc."""

    executable_file: Annotated[str, PropertyInfo(alias="executableFile")]
    """Relative path to the main executable file within the downloaded package"""

    location_hints: Annotated[List[str], PropertyInfo(alias="locationHints")]
    """Array of URIs where the Capability package can be downloaded"""

    manifest_file: Annotated[str, PropertyInfo(alias="manifestFile")]
    """Relative path to the manifest file within the downloaded package"""

    output_directory_hint: Annotated[str, PropertyInfo(alias="outputDirectoryHint")]
    """Optional hint for plugin's output/config directory"""

    summary: str
    """Brief, human-readable summary of the plugin's purpose"""


class ResourceDescriptor(BaseDescriptorParam, total=False):
    content_hash: Annotated[str, PropertyInfo(alias="contentHash")]
    """SHA256 hash of the Capability package"""

    resource_type: Annotated[
        Literal["FILE", "API", "PLUGIN", "GENERATED_DOCUMENT", "DATASET", "MODEL_ARTIFACT", "SERVICE"],
        PropertyInfo(alias="resourceType"),
    ]
    """Specific kind of resource"""

    schema: ResourceDescriptorSchema


class ToolDescriptor(BaseDescriptorParam, total=False):
    execution_pointer: Annotated[str, PropertyInfo(alias="executionPointer")]
    """Reference to a Plugin, API endpoint, or embedded logic"""

    input_schema_json: Annotated[str, PropertyInfo(alias="inputSchemaJSON")]
    """JSON Schema for tool input"""

    output_schema_json: Annotated[str, PropertyInfo(alias="outputSchemaJSON")]
    """JSON Schema for tool output"""


class PromptDescriptor(BaseDescriptorParam, total=False):
    parameters_schema_json: Annotated[str, PropertyInfo(alias="parametersSchemaJSON")]
    """JSON Schema for template parameters"""

    template: str
    """The prompt template string"""


class MemoryServiceDescriptor(BaseDescriptorParam, total=False):
    graph_schema: Annotated[str, PropertyInfo(alias="graphSchema")]
    """Describes the structure/ontology of the memory graph if applicable"""


CapabilityDescriptorParam: TypeAlias = Union[
    ResourceDescriptor, ToolDescriptor, PromptDescriptor, MemoryServiceDescriptor
]
