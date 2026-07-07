# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import List, Optional
from typing_extensions import TypeAlias

from .._models import BaseModel

__all__ = ["PeerListResponse", "PeerListResponseItem"]


class PeerListResponseItem(BaseModel):
    id: Optional[str] = None
    """Peer ID"""

    address: Optional[str] = None
    """Peer address"""

    connected_since: Optional[int] = None
    """Unix timestamp when the peer connected"""


PeerListResponse: TypeAlias = List[PeerListResponseItem]
