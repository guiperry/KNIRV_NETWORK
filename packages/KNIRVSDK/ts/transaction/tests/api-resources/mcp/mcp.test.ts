// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

const client = new KnirvchainTransactionSDK({
  apiKey: 'My API Key',
  baseURL: process.env['TEST_API_BASE_URL'] ?? 'http://127.0.0.1:4010',
});

describe('resource mcp', () => {
  // skipped: tests are disabled for the time being
  test.skip('retrieve', async () => {
    const responsePromise = client.mcp.retrieve('context_id');
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('retrieveCapabilities', async () => {
    const responsePromise = client.mcp.retrieveCapabilities();
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('retrieveCapabilities: request options and params are passed correctly', async () => {
    // ensure the request options are being passed correctly by passing an invalid HTTP method in order to cause an error
    await expect(
      client.mcp.retrieveCapabilities(
        { owner: 'owner', type: 'RESOURCE' },
        { path: '/_stainless_unknown_path' },
      ),
    ).rejects.toThrow(KnirvchainTransactionSDK.NotFoundError);
  });

  // skipped: tests are disabled for the time being
  test.skip('retrieveContexts', async () => {
    const responsePromise = client.mcp.retrieveContexts();
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('retrieveContexts: request options and params are passed correctly', async () => {
    // ensure the request options are being passed correctly by passing an invalid HTTP method in order to cause an error
    await expect(
      client.mcp.retrieveContexts(
        { capabilityId: 'capabilityId', initiator: 'initiator', interactionType: 'TOOL_INVOCATION' },
        { path: '/_stainless_unknown_path' },
      ),
    ).rejects.toThrow(KnirvchainTransactionSDK.NotFoundError);
  });
});
