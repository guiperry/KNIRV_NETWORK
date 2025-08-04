"""
Client module for the KNIRV Client SDK.

This module provides the KnirvClient class for interacting with the KNIRVCHAIN network.
"""

import asyncio
import hashlib
import logging
from typing import Dict, List, Optional, Any, Union, Tuple
import base58
import multibase
import multihash
import py_libp2p
from py_libp2p.peer.peerinfo import info_from_p2p_addr
from py_libp2p.peer.id import ID as PeerID
from py_libp2p.crypto.keys import KeyPair
from py_libp2p.crypto.rsa import create_new_key_pair
from py_libp2p.network.stream.net_stream_interface import INetStream
from py_libp2p.host.basic_host import BasicHost
from py_libp2p.network.swarm import Swarm
from py_libp2p.transport.tcp.tcp import TCP
from py_libp2p.security.insecure.transport import PLAINTEXT
from py_libp2p.kademlia.dht import KadDHT

from .parser import KnirvURI, parse_knirv_uri, KnirvURIError


class KnirvClientError(Exception):
    """Exception raised for KNIRV client errors."""
    pass


class ResourceNotFoundError(KnirvClientError):
    """Exception raised when a resource is not found."""
    pass


class ConnectionFailedError(KnirvClientError):
    """Exception raised when connection to a peer fails."""
    pass


class FetchFailedError(KnirvClientError):
    """Exception raised when fetching a resource fails."""
    pass


class ClientClosedError(KnirvClientError):
    """Exception raised when operations are attempted on a closed client."""
    pass


class ResourceData:
    """Represents data fetched from a KNIRV resource."""
    
    def __init__(self, data: bytes):
        """
        Initialize a ResourceData object.
        
        Args:
            data: The raw byte data.
        """
        self._data = data
    
    def bytes(self) -> bytes:
        """
        Get the raw byte data.
        
        Returns:
            The raw byte data.
        """
        return self._data
    
    def __str__(self) -> str:
        """
        Get the data as a string.
        
        Returns:
            The data as a string.
        """
        return self._data.decode('utf-8', errors='replace')


class KnirvClient:
    """Client for interacting with the KNIRVCHAIN network."""
    
    def __init__(self, bootstrap_peers: List[str], p2p_port: int = 0, log_enabled: bool = False):
        """
        Initialize a KnirvClient.
        
        Args:
            bootstrap_peers: List of bootstrap peer multiaddresses.
            p2p_port: Port to use for the libp2p host (0 for random).
            log_enabled: Whether to enable logging.
        """
        self.bootstrap_peers = bootstrap_peers
        self.p2p_port = p2p_port
        self.log_enabled = log_enabled
        self.closed = False
        self.bootstrapped = False
        
        # Set up logging
        self.logger = logging.getLogger("knirv_client")
        if log_enabled:
            logging.basicConfig(level=logging.INFO)
        else:
            logging.basicConfig(level=logging.WARNING)
        
        # Initialize libp2p components
        self._init_libp2p()
    
    def _init_libp2p(self):
        """Initialize libp2p components."""
        # Generate a key pair
        self.key_pair = create_new_key_pair()
        
        # Create a transport
        self.transport = TCP()
        
        # Create a security transport
        self.security = PLAINTEXT()
        
        # Create a swarm
        self.swarm = Swarm(
            peer_id=self.key_pair.get_peer_id(),
            transport=self.transport,
            security=self.security
        )
        
        # Create a host
        self.host = BasicHost(self.swarm)
        
        # Create a DHT
        self.dht = KadDHT(self.host)
        
        # Set up protocol handlers
        self._register_protocol_handlers()
        
        # Start listening
        listen_addr = f"/ip4/0.0.0.0/tcp/{self.p2p_port}"
        self.host.get_network().listen(listen_addr)
        
        if self.log_enabled:
            self.logger.info(f"KNIRV Client started with PeerID: {self.host.get_id().pretty()}")
            for addr in self.host.get_addrs():
                self.logger.info(f"Listening on: {addr}/p2p/{self.host.get_id().pretty()}")
    
    def _register_protocol_handlers(self):
        """Register protocol handlers."""
        # Register handler for the resource fetch protocol
        self.host.set_stream_handler("/knirv/resource-fetch/1.0.0", self._handle_resource_fetch)
    
    async def _handle_resource_fetch(self, stream: INetStream):
        """
        Handle incoming resource fetch requests.
        
        Args:
            stream: The network stream.
        """
        # This is a client, so we don't expect incoming resource fetch requests
        # But we need to handle them gracefully
        await stream.write(b"ERROR: This is a client node and does not serve resources")
        await stream.close()
    
    async def bootstrap(self):
        """
        Bootstrap the client by connecting to bootstrap peers and joining the DHT network.
        
        Raises:
            KnirvClientError: If bootstrapping fails.
        """
        if self.closed:
            raise ClientClosedError("Client is closed")
        
        if self.bootstrapped:
            return
        
        if self.log_enabled:
            self.logger.info("Connecting to bootstrap nodes...")
        
        # Connect to bootstrap peers
        for peer_addr in self.bootstrap_peers:
            try:
                peer_info = info_from_p2p_addr(peer_addr)
                await self.host.connect(peer_info)
                if self.log_enabled:
                    self.logger.info(f"Connected to bootstrap node: {peer_info.peer_id.pretty()}")
            except Exception as e:
                if self.log_enabled:
                    self.logger.warning(f"Error connecting to bootstrap node {peer_addr}: {e}")
        
        # Bootstrap the DHT
        try:
            await self.dht.bootstrap()
            self.bootstrapped = True
            if self.log_enabled:
                self.logger.info("DHT bootstrapped successfully")
        except Exception as e:
            raise KnirvClientError(f"Failed to bootstrap DHT: {e}")
    
    async def close(self):
        """Close the client and release resources."""
        if self.closed:
            return
        
        self.closed = True
        
        # Close the host
        await self.host.close()
        
        if self.log_enabled:
            self.logger.info("Client closed")
    
    async def fetch_resource(self, uri_string: str) -> ResourceData:
        """
        Fetch a resource from the KNIRVCHAIN network.
        
        Args:
            uri_string: The URI string of the resource to fetch.
            
        Returns:
            A ResourceData object containing the fetched data.
            
        Raises:
            KnirvURIError: If the URI is invalid.
            ResourceNotFoundError: If the resource is not found.
            ConnectionFailedError: If connection to a peer fails.
            FetchFailedError: If fetching the resource fails.
            ClientClosedError: If the client is closed.
        """
        if self.closed:
            raise ClientClosedError("Client is closed")
        
        # Ensure client is bootstrapped
        if not self.bootstrapped:
            await self.bootstrap()
        
        # Parse the URI
        uri = parse_knirv_uri(uri_string)
        
        # Find providers for the resource
        providers = await self._find_resource_providers(uri.id, uri.resource_type)
        
        # Try to connect to providers and fetch the resource
        for provider in providers:
            try:
                data = await self._fetch_from_provider(provider, uri)
                return ResourceData(data)
            except Exception as e:
                if self.log_enabled:
                    self.logger.warning(f"Failed to fetch from provider {provider.peer_id.pretty()}: {e}")
                continue
        
        raise ResourceNotFoundError(f"No providers found for resource: {uri_string}")
    
    async def _find_resource_providers(self, id_str: str, resource_type: str) -> List[Any]:
        """
        Find providers for a resource in the DHT.
        
        Args:
            id_str: The ID part of the resource.
            resource_type: The resource type part of the resource.
            
        Returns:
            A list of peer info objects for providers of the resource.
            
        Raises:
            ResourceNotFoundError: If no providers are found.
        """
        # Create a CID from the resource identifier
        resource_key = f"{id_str}.{resource_type}"
        h = hashlib.sha256()
        h.update(resource_key.encode('utf-8'))
        mh_digest = h.digest()
        
        # Create a multihash
        mh = multihash.encode(mh_digest, 'sha2-256')
        
        if self.log_enabled:
            self.logger.info(f"Looking for providers of resource: {resource_key}")
        
        # Find providers for this resource
        try:
            providers = await self.dht.find_providers(mh, count=20, timeout=30)
        except Exception as e:
            raise ResourceNotFoundError(f"Failed to find providers: {e}")
        
        if not providers:
            raise ResourceNotFoundError(f"No providers found for resource: {resource_key}")
        
        return providers
    
    async def _fetch_from_provider(self, provider: Any, uri: KnirvURI) -> bytes:
        """
        Fetch a resource from a specific provider.
        
        Args:
            provider: The peer info of the provider.
            uri: The parsed URI.
            
        Returns:
            The fetched data as bytes.
            
        Raises:
            ConnectionFailedError: If connection to the peer fails.
            FetchFailedError: If fetching the resource fails.
        """
        # Connect to the provider
        try:
            await self.host.connect(provider)
        except Exception as e:
            raise ConnectionFailedError(f"Failed to connect to peer: {e}")
        
        # Open a stream to the provider
        try:
            stream = await self.host.new_stream(provider.peer_id, ["/knirv/resource-fetch/1.0.0"])
        except Exception as e:
            raise ConnectionFailedError(f"Failed to open stream: {e}")
        
        # Prepare the request
        request = f"GET {uri.path}"
        if uri.query:
            query_params = []
            for k, values in uri.query.items():
                for v in values:
                    query_params.append(f"{k}={v}")
            request += "?" + "&".join(query_params)
        request += "\n"
        
        # Send the request
        if self.log_enabled:
            self.logger.info(f"Sending request to {provider.peer_id.pretty()}: {request}")
        
        try:
            await stream.write(request.encode('utf-8'))
        except Exception as e:
            await stream.close()
            raise FetchFailedError(f"Failed to send request: {e}")
        
        # Read the response
        try:
            response_data = await stream.read()
        except Exception as e:
            await stream.close()
            raise FetchFailedError(f"Failed to read response: {e}")
        
        # Close the stream
        await stream.close()
        
        if not response_data:
            raise FetchFailedError("Empty response from provider")
        
        # Check if the response indicates an error
        if response_data.startswith(b"ERROR:"):
            raise FetchFailedError(response_data.decode('utf-8', errors='replace'))
        
        return response_data
    
    def get_peer_id(self) -> str:
        """
        Get the peer ID of this client.
        
        Returns:
            The peer ID as a string.
        """
        return self.host.get_id().pretty()
    
    def get_multiaddrs(self) -> List[str]:
        """
        Get the multiaddresses of this client.
        
        Returns:
            A list of multiaddress strings.
        """
        addrs = []
        for addr in self.host.get_addrs():
            addrs.append(f"{addr}/p2p/{self.host.get_id().pretty()}")
        return addrs