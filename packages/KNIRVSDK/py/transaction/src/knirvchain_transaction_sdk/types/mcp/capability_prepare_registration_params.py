# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing_extensions import Required, Annotated, TypedDict

from ..._utils import PropertyInfo
from .capability_descriptor_param import CapabilityDescriptorParam

__all__ = ["CapabilityPrepareRegistrationParams"]


class CapabilityPrepareRegistrationParams(TypedDict, total=False):
    fee: Required[int]
    """Network fee for the registration transaction."""

    owner_address: Required[Annotated[str, PropertyInfo(alias="ownerAddress")]]
    """Address of the account that will own the capability."""

    descriptor: CapabilityDescriptorParam

    message: str
    """Additional information"""
