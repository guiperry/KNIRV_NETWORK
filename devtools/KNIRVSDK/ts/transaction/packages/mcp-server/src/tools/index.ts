// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import { Metadata, Endpoint, HandlerFunction } from './types';

export { Metadata, Endpoint, HandlerFunction };

import retrieve_chain from './chain/retrieve-chain';
import submit_block from './block/submit-block';
import submit_transaction from './transaction/submit-transaction';
import retrieve_txn_pool from './txn-pool/retrieve-txn-pool';
import retrieve_info from './info/retrieve-info';
import check_health from './health/check-health';
import check_ping from './ping/check-ping';
import list_peers from './peers/list-peers';
import create_uri_generator from './uri-generator/create-uri-generator';
import retrieve_mcp from './mcp/retrieve-mcp';
import retrieve_capabilities_mcp from './mcp/retrieve-capabilities-mcp';
import retrieve_contexts_mcp from './mcp/retrieve-contexts-mcp';
import retrieve_mcp_capability from './mcp/capability/retrieve-mcp-capability';
import update_mcp_capability from './mcp/capability/update-mcp-capability';
import invoke_mcp_capability from './mcp/capability/invoke-mcp-capability';
import prepare_registration_mcp_capability from './mcp/capability/prepare-registration-mcp-capability';
import retrieve_invocations_mcp_capability from './mcp/capability/retrieve-invocations-mcp-capability';

export const endpoints: Endpoint[] = [];

function addEndpoint(endpoint: Endpoint) {
  endpoints.push(endpoint);
}

addEndpoint(retrieve_chain);
addEndpoint(submit_block);
addEndpoint(submit_transaction);
addEndpoint(retrieve_txn_pool);
addEndpoint(retrieve_info);
addEndpoint(check_health);
addEndpoint(check_ping);
addEndpoint(list_peers);
addEndpoint(create_uri_generator);
addEndpoint(retrieve_mcp);
addEndpoint(retrieve_capabilities_mcp);
addEndpoint(retrieve_contexts_mcp);
addEndpoint(retrieve_mcp_capability);
addEndpoint(update_mcp_capability);
addEndpoint(invoke_mcp_capability);
addEndpoint(prepare_registration_mcp_capability);
addEndpoint(retrieve_invocations_mcp_capability);

export type Filter = {
  type: 'resource' | 'operation' | 'tag' | 'tool';
  op: 'include' | 'exclude';
  value: string;
};

export function query(filters: Filter[], endpoints: Endpoint[]): Endpoint[] {
  const allExcludes = filters.length > 0 && filters.every((filter) => filter.op === 'exclude');
  const unmatchedFilters = new Set(filters);

  const filtered = endpoints.filter((endpoint: Endpoint) => {
    let included = false || allExcludes;

    for (const filter of filters) {
      if (match(filter, endpoint)) {
        unmatchedFilters.delete(filter);
        included = filter.op === 'include';
      }
    }

    return included;
  });

  // Check if any filters didn't match
  const unmatched = Array.from(unmatchedFilters).filter((f) => f.type === 'tool' || f.type === 'resource');
  if (unmatched.length > 0) {
    throw new Error(
      `The following filters did not match any endpoints: ${unmatched
        .map((f) => `${f.type}=${f.value}`)
        .join(', ')}`,
    );
  }

  return filtered;
}

function match({ type, value }: Filter, endpoint: Endpoint): boolean {
  switch (type) {
    case 'resource': {
      const regexStr = '^' + normalizeResource(value).replace(/\*/g, '.*') + '$';
      const regex = new RegExp(regexStr);
      return regex.test(normalizeResource(endpoint.metadata.resource));
    }
    case 'operation':
      return endpoint.metadata.operation === value;
    case 'tag':
      return endpoint.metadata.tags.includes(value);
    case 'tool':
      return endpoint.tool.name === value;
  }
}

function normalizeResource(resource: string): string {
  return resource.toLowerCase().replace(/[^a-z.*\-_]*/g, '');
}
