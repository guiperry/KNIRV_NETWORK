# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing_extensions import Literal

import httpx

from ...types import mcp_retrieve_contexts_params, mcp_retrieve_capabilities_params
from ..._types import NOT_GIVEN, Body, Query, Headers, NotGiven
from ..._utils import maybe_transform, async_maybe_transform
from ..._compat import cached_property
from .capability import (
    CapabilityResource,
    AsyncCapabilityResource,
    CapabilityResourceWithRawResponse,
    AsyncCapabilityResourceWithRawResponse,
    CapabilityResourceWithStreamingResponse,
    AsyncCapabilityResourceWithStreamingResponse,
)
from ..._resource import SyncAPIResource, AsyncAPIResource
from ..._response import (
    to_raw_response_wrapper,
    to_streamed_response_wrapper,
    async_to_raw_response_wrapper,
    async_to_streamed_response_wrapper,
)
from ..._base_client import make_request_options
from ...types.context_record import ContextRecord
from ...types.mcp_retrieve_contexts_response import McpRetrieveContextsResponse
from ...types.mcp_retrieve_capabilities_response import McpRetrieveCapabilitiesResponse

__all__ = ["McpResource", "AsyncMcpResource"]


class McpResource(SyncAPIResource):
    @cached_property
    def capability(self) -> CapabilityResource:
        return CapabilityResource(self._client)

    @cached_property
    def with_raw_response(self) -> McpResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return McpResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> McpResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return McpResourceWithStreamingResponse(self)

    def retrieve(
        self,
        context_id: str,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> ContextRecord:
        """
        Retrieves a specific context record by ID

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        if not context_id:
            raise ValueError(f"Expected a non-empty value for `context_id` but received {context_id!r}")
        return self._get(
            f"/mcp/context/{context_id}",
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=ContextRecord,
        )

    def retrieve_capabilities(
        self,
        *,
        owner: str | NotGiven = NOT_GIVEN,
        type: Literal["RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE"] | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> McpRetrieveCapabilitiesResponse:
        """
        Lists all registered capabilities

        Args:
          owner: Filter by owner address

          type: Filter by capability type

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._get(
            "/mcp/capabilities",
            options=make_request_options(
                extra_headers=extra_headers,
                extra_query=extra_query,
                extra_body=extra_body,
                timeout=timeout,
                query=maybe_transform(
                    {
                        "owner": owner,
                        "type": type,
                    },
                    mcp_retrieve_capabilities_params.McpRetrieveCapabilitiesParams,
                ),
            ),
            cast_to=McpRetrieveCapabilitiesResponse,
        )

    def retrieve_contexts(
        self,
        *,
        capability_id: str | NotGiven = NOT_GIVEN,
        initiator: str | NotGiven = NOT_GIVEN,
        interaction_type: Literal[
            "TOOL_INVOCATION",
            "PROMPT_USAGE",
            "RESOURCE_ACCESS",
            "PLUGIN_EXECUTION",
            "SAMPLING_REQUEST_SENT",
            "SAMPLING_RESPONSE_RECEIVED",
            "MEMORY_WRITE",
            "CAPABILITY_REGISTRATION",
        ]
        | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> McpRetrieveContextsResponse:
        """
        Lists all context records

        Args:
          capability_id: Filter by capability ID

          initiator: Filter by initiator address

          interaction_type: Filter by interaction type

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._get(
            "/mcp/contexts",
            options=make_request_options(
                extra_headers=extra_headers,
                extra_query=extra_query,
                extra_body=extra_body,
                timeout=timeout,
                query=maybe_transform(
                    {
                        "capability_id": capability_id,
                        "initiator": initiator,
                        "interaction_type": interaction_type,
                    },
                    mcp_retrieve_contexts_params.McpRetrieveContextsParams,
                ),
            ),
            cast_to=McpRetrieveContextsResponse,
        )


class AsyncMcpResource(AsyncAPIResource):
    @cached_property
    def capability(self) -> AsyncCapabilityResource:
        return AsyncCapabilityResource(self._client)

    @cached_property
    def with_raw_response(self) -> AsyncMcpResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return AsyncMcpResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> AsyncMcpResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return AsyncMcpResourceWithStreamingResponse(self)

    async def retrieve(
        self,
        context_id: str,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> ContextRecord:
        """
        Retrieves a specific context record by ID

        Args:
          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        if not context_id:
            raise ValueError(f"Expected a non-empty value for `context_id` but received {context_id!r}")
        return await self._get(
            f"/mcp/context/{context_id}",
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=ContextRecord,
        )

    async def retrieve_capabilities(
        self,
        *,
        owner: str | NotGiven = NOT_GIVEN,
        type: Literal["RESOURCE", "TOOL", "PROMPT", "MEMORY_SERVICE"] | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> McpRetrieveCapabilitiesResponse:
        """
        Lists all registered capabilities

        Args:
          owner: Filter by owner address

          type: Filter by capability type

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._get(
            "/mcp/capabilities",
            options=make_request_options(
                extra_headers=extra_headers,
                extra_query=extra_query,
                extra_body=extra_body,
                timeout=timeout,
                query=await async_maybe_transform(
                    {
                        "owner": owner,
                        "type": type,
                    },
                    mcp_retrieve_capabilities_params.McpRetrieveCapabilitiesParams,
                ),
            ),
            cast_to=McpRetrieveCapabilitiesResponse,
        )

    async def retrieve_contexts(
        self,
        *,
        capability_id: str | NotGiven = NOT_GIVEN,
        initiator: str | NotGiven = NOT_GIVEN,
        interaction_type: Literal[
            "TOOL_INVOCATION",
            "PROMPT_USAGE",
            "RESOURCE_ACCESS",
            "PLUGIN_EXECUTION",
            "SAMPLING_REQUEST_SENT",
            "SAMPLING_RESPONSE_RECEIVED",
            "MEMORY_WRITE",
            "CAPABILITY_REGISTRATION",
        ]
        | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> McpRetrieveContextsResponse:
        """
        Lists all context records

        Args:
          capability_id: Filter by capability ID

          initiator: Filter by initiator address

          interaction_type: Filter by interaction type

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._get(
            "/mcp/contexts",
            options=make_request_options(
                extra_headers=extra_headers,
                extra_query=extra_query,
                extra_body=extra_body,
                timeout=timeout,
                query=await async_maybe_transform(
                    {
                        "capability_id": capability_id,
                        "initiator": initiator,
                        "interaction_type": interaction_type,
                    },
                    mcp_retrieve_contexts_params.McpRetrieveContextsParams,
                ),
            ),
            cast_to=McpRetrieveContextsResponse,
        )


class McpResourceWithRawResponse:
    def __init__(self, mcp: McpResource) -> None:
        self._mcp = mcp

        self.retrieve = to_raw_response_wrapper(
            mcp.retrieve,
        )
        self.retrieve_capabilities = to_raw_response_wrapper(
            mcp.retrieve_capabilities,
        )
        self.retrieve_contexts = to_raw_response_wrapper(
            mcp.retrieve_contexts,
        )

    @cached_property
    def capability(self) -> CapabilityResourceWithRawResponse:
        return CapabilityResourceWithRawResponse(self._mcp.capability)


class AsyncMcpResourceWithRawResponse:
    def __init__(self, mcp: AsyncMcpResource) -> None:
        self._mcp = mcp

        self.retrieve = async_to_raw_response_wrapper(
            mcp.retrieve,
        )
        self.retrieve_capabilities = async_to_raw_response_wrapper(
            mcp.retrieve_capabilities,
        )
        self.retrieve_contexts = async_to_raw_response_wrapper(
            mcp.retrieve_contexts,
        )

    @cached_property
    def capability(self) -> AsyncCapabilityResourceWithRawResponse:
        return AsyncCapabilityResourceWithRawResponse(self._mcp.capability)


class McpResourceWithStreamingResponse:
    def __init__(self, mcp: McpResource) -> None:
        self._mcp = mcp

        self.retrieve = to_streamed_response_wrapper(
            mcp.retrieve,
        )
        self.retrieve_capabilities = to_streamed_response_wrapper(
            mcp.retrieve_capabilities,
        )
        self.retrieve_contexts = to_streamed_response_wrapper(
            mcp.retrieve_contexts,
        )

    @cached_property
    def capability(self) -> CapabilityResourceWithStreamingResponse:
        return CapabilityResourceWithStreamingResponse(self._mcp.capability)


class AsyncMcpResourceWithStreamingResponse:
    def __init__(self, mcp: AsyncMcpResource) -> None:
        self._mcp = mcp

        self.retrieve = async_to_streamed_response_wrapper(
            mcp.retrieve,
        )
        self.retrieve_capabilities = async_to_streamed_response_wrapper(
            mcp.retrieve_capabilities,
        )
        self.retrieve_contexts = async_to_streamed_response_wrapper(
            mcp.retrieve_contexts,
        )

    @cached_property
    def capability(self) -> AsyncCapabilityResourceWithStreamingResponse:
        return AsyncCapabilityResourceWithStreamingResponse(self._mcp.capability)
