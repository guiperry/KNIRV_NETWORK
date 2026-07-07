# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

import os
from typing import Any, cast

import pytest

from tests.utils import assert_matches_type
from knirvchain_transaction_sdk import KnirvchainTransactionSDK, AsyncKnirvchainTransactionSDK
from knirvchain_transaction_sdk.types import UriGeneratorCreateResponse

base_url = os.environ.get("TEST_API_BASE_URL", "http://127.0.0.1:4010")


class TestUriGenerator:
    parametrize = pytest.mark.parametrize("client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    def test_method_create(self, client: KnirvchainTransactionSDK) -> None:
        uri_generator = client.uri_generator.create()
        assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_method_create_with_all_params(self, client: KnirvchainTransactionSDK) -> None:
        uri_generator = client.uri_generator.create(
            content_hash="content_hash",
            metadata={"foo": "bar"},
            owner="owner",
            resource_type="resource_type",
        )
        assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_create(self, client: KnirvchainTransactionSDK) -> None:
        response = client.uri_generator.with_raw_response.create()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        uri_generator = response.parse()
        assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_create(self, client: KnirvchainTransactionSDK) -> None:
        with client.uri_generator.with_streaming_response.create() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            uri_generator = response.parse()
            assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

        assert cast(Any, response.is_closed) is True


class TestAsyncUriGenerator:
    parametrize = pytest.mark.parametrize("async_client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_create(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        uri_generator = await async_client.uri_generator.create()
        assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_create_with_all_params(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        uri_generator = await async_client.uri_generator.create(
            content_hash="content_hash",
            metadata={"foo": "bar"},
            owner="owner",
            resource_type="resource_type",
        )
        assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_create(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.uri_generator.with_raw_response.create()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        uri_generator = await response.parse()
        assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_create(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.uri_generator.with_streaming_response.create() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            uri_generator = await response.parse()
            assert_matches_type(UriGeneratorCreateResponse, uri_generator, path=["response"])

        assert cast(Any, response.is_closed) is True
