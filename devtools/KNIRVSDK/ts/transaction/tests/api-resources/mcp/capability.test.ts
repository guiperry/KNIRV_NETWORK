// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

const client = new KnirvchainTransactionSDK({
  apiKey: 'My API Key',
  baseURL: process.env['TEST_API_BASE_URL'] ?? 'http://127.0.0.1:4010',
});

describe('resource capability', () => {
  // skipped: tests are disabled for the time being
  test.skip('retrieve', async () => {
    const responsePromise = client.mcp.capability.retrieve('capability_id');
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('update: only required params', async () => {
    const responsePromise = client.mcp.capability.update({
      signedTransaction: {
        id: '0xabcdef123456...',
        fee: 'fee',
        from: 'from',
        public_key: 'public_key',
        signature: 'U3RhaW5sZXNzIHJvY2tz',
        timestamp: 0,
        type: 'type',
        version: 1,
      },
    });
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('update: required and optional params', async () => {
    const response = await client.mcp.capability.update({
      signedTransaction: {
        id: '0xabcdef123456...',
        fee: 'fee',
        from: 'from',
        public_key: 'public_key',
        signature: 'U3RhaW5sZXNzIHJvY2tz',
        timestamp: 0,
        type: 'type',
        version: 1,
        data: { foo: 'bar' },
        status: 'status',
        to: 'to',
        transaction_hash: 'transaction_hash',
        value: '1000000000000000000',
      },
    });
  });

  // skipped: tests are disabled for the time being
  test.skip('invoke: only required params', async () => {
    const responsePromise = client.mcp.capability.invoke({
      signedTransaction: {
        id: '0xabcdef123456...',
        fee: 'fee',
        from: 'from',
        public_key: 'public_key',
        signature: 'U3RhaW5sZXNzIHJvY2tz',
        timestamp: 0,
        type: 'type',
        version: 1,
      },
    });
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('invoke: required and optional params', async () => {
    const response = await client.mcp.capability.invoke({
      signedTransaction: {
        id: '0xabcdef123456...',
        fee: 'fee',
        from: 'from',
        public_key: 'public_key',
        signature: 'U3RhaW5sZXNzIHJvY2tz',
        timestamp: 0,
        type: 'type',
        version: 1,
        data: { foo: 'bar' },
        status: 'status',
        to: 'to',
        transaction_hash: 'transaction_hash',
        value: '1000000000000000000',
      },
    });
  });

  // skipped: tests are disabled for the time being
  test.skip('prepareRegistration: only required params', async () => {
    const responsePromise = client.mcp.capability.prepareRegistration({
      fee: 0,
      ownerAddress: 'ownerAddress',
    });
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('prepareRegistration: required and optional params', async () => {
    const response = await client.mcp.capability.prepareRegistration({
      fee: 0,
      ownerAddress: 'ownerAddress',
      descriptor: {
        id: 'id',
        capabilityType: 'RESOURCE',
        customMetadata: { foo: 'bar' },
        description: 'description',
        gasFeeNRN: 0,
        name: 'name',
        owner: 'owner',
        timestamp: 0,
        version: 'version',
        contentHash: 'contentHash',
        resourceType: 'FILE',
        schema: {
          accessInfo: { foo: 'bar' },
          executableFile: 'executableFile',
          locationHints: ['string'],
          manifestFile: 'manifestFile',
          outputDirectoryHint: 'outputDirectoryHint',
          summary: 'summary',
        },
      },
      message: 'message',
    });
  });

  // skipped: tests are disabled for the time being
  test.skip('retrieveInvocations', async () => {
    const responsePromise = client.mcp.capability.retrieveInvocations('capability_id');
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });
});
