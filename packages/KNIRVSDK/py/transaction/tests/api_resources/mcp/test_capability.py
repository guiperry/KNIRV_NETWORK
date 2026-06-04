# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

import os
from typing import Any, cast

import pytest

from tests.utils import assert_matches_type
from knirvchain_transaction_sdk import KnirvchainTransactionSDK, AsyncKnirvchainTransactionSDK
from knirvchain_transaction_sdk.types.mcp import (
    CapabilityDescriptor,
    CapabilityInvokeResponse,
    CapabilityUpdateResponse,
    CapabilityPrepareRegistrationResponse,
    CapabilityRetrieveInvocationsResponse,
)

base_url = os.environ.get("TEST_API_BASE_URL", "http://127.0.0.1:4010")


class TestCapability:
    parametrize = pytest.mark.parametrize("client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.retrieve(
            "capability_id",
        )
        assert_matches_type(CapabilityDescriptor, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.capability.with_raw_response.retrieve(
            "capability_id",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = response.parse()
        assert_matches_type(CapabilityDescriptor, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.capability.with_streaming_response.retrieve(
            "capability_id",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = response.parse()
            assert_matches_type(CapabilityDescriptor, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    def test_path_params_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        with pytest.raises(ValueError, match=r"Expected a non-empty value for `capability_id` but received ''"):
            client.mcp.capability.with_raw_response.retrieve(
                "",
            )

    @pytest.mark.skip()
    @parametrize
    def test_method_update(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )
        assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_method_update_with_all_params(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
                "data": {"foo": "bar"},
                "status": "status",
                "to": "to",
                "transaction_hash": "transaction_hash",
                "value": "1000000000000000000",
            },
        )
        assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_update(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.capability.with_raw_response.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = response.parse()
        assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_update(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.capability.with_streaming_response.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = response.parse()
            assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    def test_method_invoke(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )
        assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_method_invoke_with_all_params(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
                "data": {"foo": "bar"},
                "status": "status",
                "to": "to",
                "transaction_hash": "transaction_hash",
                "value": "1000000000000000000",
            },
        )
        assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_invoke(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.capability.with_raw_response.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = response.parse()
        assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_invoke(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.capability.with_streaming_response.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = response.parse()
            assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    def test_method_prepare_registration(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
        )
        assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_method_prepare_registration_with_all_params(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
            descriptor={
                "id": "id",
                "capability_type": "RESOURCE",
                "custom_metadata": {"foo": "bar"},
                "description": "description",
                "gas_fee_nrn": 0,
                "name": "name",
                "owner": "owner",
                "timestamp": 0,
                "version": "version",
                "content_hash": "contentHash",
                "resource_type": "FILE",
                "schema": {
                    "access_info": {"foo": "bar"},
                    "executable_file": "executableFile",
                    "location_hints": ["string"],
                    "manifest_file": "manifestFile",
                    "output_directory_hint": "outputDirectoryHint",
                    "summary": "summary",
                },
            },
            message="message",
        )
        assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_prepare_registration(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.capability.with_raw_response.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = response.parse()
        assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_prepare_registration(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.capability.with_streaming_response.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = response.parse()
            assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve_invocations(self, client: KnirvchainTransactionSDK) -> None:
        capability = client.mcp.capability.retrieve_invocations(
            "capability_id",
        )
        assert_matches_type(CapabilityRetrieveInvocationsResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_retrieve_invocations(self, client: KnirvchainTransactionSDK) -> None:
        response = client.mcp.capability.with_raw_response.retrieve_invocations(
            "capability_id",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = response.parse()
        assert_matches_type(CapabilityRetrieveInvocationsResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_retrieve_invocations(self, client: KnirvchainTransactionSDK) -> None:
        with client.mcp.capability.with_streaming_response.retrieve_invocations(
            "capability_id",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = response.parse()
            assert_matches_type(CapabilityRetrieveInvocationsResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    def test_path_params_retrieve_invocations(self, client: KnirvchainTransactionSDK) -> None:
        with pytest.raises(ValueError, match=r"Expected a non-empty value for `capability_id` but received ''"):
            client.mcp.capability.with_raw_response.retrieve_invocations(
                "",
            )


class TestAsyncCapability:
    parametrize = pytest.mark.parametrize("async_client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        capability = await async_client.mcp.capability.retrieve(
            "capability_id",
        )
        assert_matches_type(CapabilityDescriptor, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.capability.with_raw_response.retrieve(
            "capability_id",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = await response.parse()
        assert_matches_type(CapabilityDescriptor, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.capability.with_streaming_response.retrieve(
            "capability_id",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = await response.parse()
            assert_matches_type(CapabilityDescriptor, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    async def test_path_params_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        with pytest.raises(ValueError, match=r"Expected a non-empty value for `capability_id` but received ''"):
            await async_client.mcp.capability.with_raw_response.retrieve(
                "",
            )

    @pytest.mark.skip()
    @parametrize
    async def test_method_update(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        capability = await async_client.mcp.capability.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )
        assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_update_with_all_params(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        capability = await async_client.mcp.capability.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
                "data": {"foo": "bar"},
                "status": "status",
                "to": "to",
                "transaction_hash": "transaction_hash",
                "value": "1000000000000000000",
            },
        )
        assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_update(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.capability.with_raw_response.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = await response.parse()
        assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_update(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.capability.with_streaming_response.update(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = await response.parse()
            assert_matches_type(CapabilityUpdateResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    async def test_method_invoke(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        capability = await async_client.mcp.capability.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )
        assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_invoke_with_all_params(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        capability = await async_client.mcp.capability.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
                "data": {"foo": "bar"},
                "status": "status",
                "to": "to",
                "transaction_hash": "transaction_hash",
                "value": "1000000000000000000",
            },
        )
        assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_invoke(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.capability.with_raw_response.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = await response.parse()
        assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_invoke(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.capability.with_streaming_response.invoke(
            signed_transaction={
                "id": "0xabcdef123456...",
                "fee": "fee",
                "from": "from",
                "public_key": "public_key",
                "signature": "U3RhaW5sZXNzIHJvY2tz",
                "timestamp": 0,
                "type": "type",
                "version": 1,
            },
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = await response.parse()
            assert_matches_type(CapabilityInvokeResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    async def test_method_prepare_registration(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        capability = await async_client.mcp.capability.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
        )
        assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_prepare_registration_with_all_params(
        self, async_client: AsyncKnirvchainTransactionSDK
    ) -> None:
        capability = await async_client.mcp.capability.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
            descriptor={
                "id": "id",
                "capability_type": "RESOURCE",
                "custom_metadata": {"foo": "bar"},
                "description": "description",
                "gas_fee_nrn": 0,
                "name": "name",
                "owner": "owner",
                "timestamp": 0,
                "version": "version",
                "content_hash": "contentHash",
                "resource_type": "FILE",
                "schema": {
                    "access_info": {"foo": "bar"},
                    "executable_file": "executableFile",
                    "location_hints": ["string"],
                    "manifest_file": "manifestFile",
                    "output_directory_hint": "outputDirectoryHint",
                    "summary": "summary",
                },
            },
            message="message",
        )
        assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_prepare_registration(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.capability.with_raw_response.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = await response.parse()
        assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_prepare_registration(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.capability.with_streaming_response.prepare_registration(
            fee=0,
            owner_address="ownerAddress",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = await response.parse()
            assert_matches_type(CapabilityPrepareRegistrationResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve_invocations(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        capability = await async_client.mcp.capability.retrieve_invocations(
            "capability_id",
        )
        assert_matches_type(CapabilityRetrieveInvocationsResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_retrieve_invocations(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.mcp.capability.with_raw_response.retrieve_invocations(
            "capability_id",
        )

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        capability = await response.parse()
        assert_matches_type(CapabilityRetrieveInvocationsResponse, capability, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_retrieve_invocations(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.mcp.capability.with_streaming_response.retrieve_invocations(
            "capability_id",
        ) as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            capability = await response.parse()
            assert_matches_type(CapabilityRetrieveInvocationsResponse, capability, path=["response"])

        assert cast(Any, response.is_closed) is True

    @pytest.mark.skip()
    @parametrize
    async def test_path_params_retrieve_invocations(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        with pytest.raises(ValueError, match=r"Expected a non-empty value for `capability_id` but received ''"):
            await async_client.mcp.capability.with_raw_response.retrieve_invocations(
                "",
            )
