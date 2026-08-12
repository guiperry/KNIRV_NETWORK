import {
  marshalWasmPublicationPayload,
  parseWasmPublicationPayload,
  marshalWasmManifestPayload,
  parseWasmManifestPayload,
  WasmPublicationPayload,
  WasmManifestPayload,
} from './wasm';
import { marshalRelayEnvelope, parseRelayEnvelope, RelayEnvelope } from './relay';

// Cross-language golden vectors: KNIRV_NETWORK/packages/KNIRVSDK/testvectors/wasm_payloads.json.
// Both this file and go/signing/wasm_test.go must derive canonical_hex byte-for-byte from the
// same input — see KNIRV_CORP/packages/controller/stateless_pwa_controller.md section 3.4.
// eslint-disable-next-line @typescript-eslint/no-var-requires
const vectors = require('../../../testvectors/wasm_payloads.json');

function toHex(bytes: Uint8Array): string {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}

describe('wasm canonical payloads — golden vectors', () => {
  it('matches the Go-derived wasm_publication_v1 canonical bytes', () => {
    const input = vectors.wasm_publication_v1.input;
    const payload: WasmPublicationPayload = {
      networkId: input.network_id,
      networkFingerprint: input.network_fingerprint,
      artifactDigest: input.artifact_digest,
      byteSize: input.byte_size,
      moduleKind: input.module_kind,
      abiVersion: input.abi_version,
      moduleSchemaVersion: input.module_schema_version,
      buildId: input.build_id,
      toolchainDigest: input.toolchain_digest,
      selfTestDigest: input.self_test_digest,
      publisherAddress: input.publisher_address,
      dveTemplateId: input.dve_template_id,
    };

    const bytes = marshalWasmPublicationPayload(payload);
    expect(toHex(bytes)).toBe(vectors.wasm_publication_v1.canonical_hex);

    const roundTrip = parseWasmPublicationPayload(bytes);
    expect(roundTrip.artifactDigest).toBe(input.artifact_digest);
    expect(roundTrip.publisherAddress).toBe(input.publisher_address);
    expect(String(roundTrip.byteSize)).toBe(String(input.byte_size));
  });

  it('matches the Go-derived wasm_manifest_v1 canonical bytes', () => {
    const input = vectors.wasm_manifest_v1.input;
    const payload: WasmManifestPayload = {
      manifestId: input.manifest_id,
      networkId: input.network_id,
      chainId: input.chain_id,
      networkFingerprint: input.network_fingerprint,
      userSubject: input.user_subject,
      deviceId: input.device_id,
      dveId: input.dve_id,
      leaseEpoch: input.lease_epoch,
      modules: input.modules.map((m: any) => ({
        moduleKind: m.module_kind,
        artifactDigest: m.artifact_digest,
        byteSize: m.byte_size,
        abiVersion: m.abi_version,
        moduleSchemaVersion: m.module_schema_version,
        capabilitiesJson: m.capabilities_json,
        configurationDigest: m.configuration_digest,
        downloadPath: m.download_path,
        publisherAddress: m.publisher_address,
        publicationStatementDigest: m.publication_statement_digest,
      })),
      relayTargetType: input.relay_target_type,
      relayTargetId: input.relay_target_id,
      assignmentId: input.assignment_id,
      assignmentVersion: input.assignment_version,
      supersedesAssignmentId: input.supersedes_assignment_id,
    };

    const bytes = marshalWasmManifestPayload(payload);
    expect(toHex(bytes)).toBe(vectors.wasm_manifest_v1.canonical_hex);

    const roundTrip = parseWasmManifestPayload(bytes);
    expect(roundTrip.modules).toHaveLength(input.modules.length);
    expect(roundTrip.assignmentId).toBe(input.assignment_id);
    expect(roundTrip.modules[0].artifactDigest).toBe(input.modules[0].artifact_digest);
  });

  it('matches the Go-derived relay_envelope_v1 canonical bytes', () => {
    const input = vectors.relay_envelope_v1.input;
    const envelope: RelayEnvelope = {
      requestId: input.request_id,
      userSubject: input.user_subject,
      deviceId: input.device_id,
      dveId: input.dve_id,
      targetType: input.target_type,
      targetId: input.target_id,
      capability: input.capability,
      sequence: input.sequence,
      leaseEpoch: input.lease_epoch,
      issuedAtUnix: input.issued_at_unix,
      expiresAtUnix: input.expires_at_unix,
      payloadDigest: input.payload_digest,
    };

    const bytes = marshalRelayEnvelope(envelope);
    expect(toHex(bytes)).toBe(vectors.relay_envelope_v1.canonical_hex);

    const roundTrip = parseRelayEnvelope(bytes);
    expect(roundTrip.requestId).toBe(input.request_id);
    expect(roundTrip.payloadDigest).toBe(input.payload_digest);
    expect(String(roundTrip.sequence)).toBe(String(input.sequence));
  });
});
