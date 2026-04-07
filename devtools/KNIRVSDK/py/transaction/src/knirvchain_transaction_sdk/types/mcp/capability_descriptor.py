# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Dict, List, Union, Optional
from typing_extensions import Literal, TypeAlias

from pydantic import Field as FieldInfo

from ..._models import BaseModel
from .base_descriptor import BaseDescriptor

__all__ = [
    "CapabilityDescriptor",
    "ResourceDescriptor",
    "ResourceDescriptorSchema",
    "ToolDescriptor",
    "PromptDescriptor",
    "MemoryServiceDescriptor",
]


class ResourceDescriptorSchema(BaseModel):
    access_info: Optional[Dict[str, object]] = FieldInfo(alias="accessInfo", default=None)
    """For APIs - endpoint, auth details, etc."""

    executable_file: Optional[str] = FieldInfo(alias="executableFile", default=None)
    """Relative path to the main executable file within the downloaded package"""

    location_hints: Optional[List[str]] = FieldInfo(alias="locationHints", default=None)
    """Array of URIs where the Capability package can be downloaded"""

    manifest_file: Optional[str] = FieldInfo(alias="manifestFile", default=None)
    """Relative path to the manifest file within the downloaded package"""

    output_directory_hint: Optional[str] = FieldInfo(alias="outputDirectoryHint", default=None)
    """Optional hint for plugin's output/config directory"""

    summary: Optional[str] = None
    """Brief, human-readable summary of the plugin's purpose"""


class ResourceDescriptor(BaseDescriptor):
    content_hash: Optional[str] = FieldInfo(alias="contentHash", default=None)
    """SHA256 hash of the Capability package"""

    resource_type: Optional[
        Literal["FILE", "API", "PLUGIN", "GENERATED_DOCUMENT", "DATASET", "MODEL_ARTIFACT", "SERVICE"]
    ] = FieldInfo(alias="resourceType", default=None)
    """Specific kind of resource"""

    schema_: Optional[ResourceDescriptorSchema] = FieldInfo(alias="schema", default=None)


class ToolDescriptor(BaseDescriptor):
    execution_pointer: Optional[str] = FieldInfo(alias="executionPointer", default=None)
    """Reference to a Plugin, API endpoint, or embedded logic"""

    input_schema_json: Optional[str] = FieldInfo(alias="inputSchemaJSON", default=None)
    """JSON Schema for tool input"""

    output_schema_json: Optional[str] = FieldInfo(alias="outputSchemaJSON", default=None)
    """JSON Schema for tool output"""


class PromptDescriptor(BaseDescriptor):
    parameters_schema_json: Optional[str] = FieldInfo(alias="parametersSchemaJSON", default=None)
    """JSON Schema for template parameters"""

    template: Optional[str] = None
    """The prompt template string"""


class MemoryServiceDescriptor(BaseDescriptor):
    graph_schema: Optional[str] = FieldInfo(alias="graphSchema", default=None)
    """Describes the structure/ontology of the memory graph if applicable"""


CapabilityDescriptor: TypeAlias = Union[ResourceDescriptor, ToolDescriptor, PromptDescriptor, MemoryServiceDescriptor]
