/* tslint:disable */
/* eslint-disable */
export function main(): void;
export function get_version(): string;
export function get_build_info(): string;
export class EmbeddedKNIRVChain {
  free(): void;
  constructor();
  /**
   * Initialize the Revolutionary Embedded KNIRVCHAIN
   */
  initialize(): void;
  /**
   * Register a new LoRA Adapter Skill
   */
  register_skill(skill: LoRAAdapterSkill): void;
  /**
   * Revolutionary Skill Invocation via WASM
   */
  invoke_skill(request: SkillInvocationRequest): SkillInvocationResponse;
  /**
   * Get skill count
   */
  get_skill_count(): number;
  /**
   * Check if initialized
   */
  is_initialized(): boolean;
}
export class ErrorContext {
  free(): void;
  constructor(agent_id: string, error_type: string, error_message: string, task_description: string);
  timestamp: bigint;
}
export class LoRAAdapterSkill {
  free(): void;
  constructor(skill_id: string, skill_name: string, description: string, base_model_compatibility: string, version: number, rank: number, alpha: number);
  version: number;
  rank: number;
  alpha: number;
  readonly skill_id: string;
  readonly skill_name: string;
  readonly description: string;
}
export class SkillInvocationRequest {
  free(): void;
  constructor(invocation_id: string, agent_id: string, nrn_token: string, skill_uri: string);
  timestamp: bigint;
}
export class SkillInvocationResponse {
  free(): void;
  constructor(invocation_id: string, status: string, execution_time: bigint, skill_data: string);
  execution_time: bigint;
  memory_used: bigint;
  consensus_reached: boolean;
  readonly invocation_id: string;
  readonly status: string;
  readonly skill_data: string;
}

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;

export interface InitOutput {
  readonly memory: WebAssembly.Memory;
  readonly main: () => void;
  readonly __wbg_loraadapterskill_free: (a: number, b: number) => void;
  readonly __wbg_get_loraadapterskill_version: (a: number) => number;
  readonly __wbg_set_loraadapterskill_version: (a: number, b: number) => void;
  readonly __wbg_get_loraadapterskill_rank: (a: number) => number;
  readonly __wbg_set_loraadapterskill_rank: (a: number, b: number) => void;
  readonly __wbg_get_loraadapterskill_alpha: (a: number) => number;
  readonly __wbg_set_loraadapterskill_alpha: (a: number, b: number) => void;
  readonly loraadapterskill_new: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number, i: number, j: number, k: number) => number;
  readonly loraadapterskill_skill_id: (a: number) => [number, number];
  readonly loraadapterskill_skill_name: (a: number) => [number, number];
  readonly loraadapterskill_description: (a: number) => [number, number];
  readonly __wbg_errorcontext_free: (a: number, b: number) => void;
  readonly __wbg_get_errorcontext_timestamp: (a: number) => bigint;
  readonly __wbg_set_errorcontext_timestamp: (a: number, b: bigint) => void;
  readonly errorcontext_new: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number) => number;
  readonly __wbg_skillinvocationrequest_free: (a: number, b: number) => void;
  readonly skillinvocationrequest_new: (a: number, b: number, c: number, d: number, e: number, f: number, g: number, h: number) => number;
  readonly __wbg_skillinvocationresponse_free: (a: number, b: number) => void;
  readonly __wbg_get_skillinvocationresponse_memory_used: (a: number) => bigint;
  readonly __wbg_set_skillinvocationresponse_memory_used: (a: number, b: bigint) => void;
  readonly __wbg_get_skillinvocationresponse_consensus_reached: (a: number) => number;
  readonly __wbg_set_skillinvocationresponse_consensus_reached: (a: number, b: number) => void;
  readonly skillinvocationresponse_new: (a: number, b: number, c: number, d: number, e: bigint, f: number, g: number) => number;
  readonly skillinvocationresponse_invocation_id: (a: number) => [number, number];
  readonly skillinvocationresponse_status: (a: number) => [number, number];
  readonly skillinvocationresponse_skill_data: (a: number) => [number, number];
  readonly __wbg_embeddedknirvchain_free: (a: number, b: number) => void;
  readonly embeddedknirvchain_new: () => number;
  readonly embeddedknirvchain_initialize: (a: number) => [number, number];
  readonly embeddedknirvchain_register_skill: (a: number, b: number) => [number, number];
  readonly embeddedknirvchain_invoke_skill: (a: number, b: number) => [number, number, number];
  readonly embeddedknirvchain_get_skill_count: (a: number) => number;
  readonly embeddedknirvchain_is_initialized: (a: number) => number;
  readonly get_version: () => [number, number];
  readonly get_build_info: () => [number, number];
  readonly __wbg_get_skillinvocationrequest_timestamp: (a: number) => bigint;
  readonly __wbg_get_skillinvocationresponse_execution_time: (a: number) => bigint;
  readonly __wbg_set_skillinvocationrequest_timestamp: (a: number, b: bigint) => void;
  readonly __wbg_set_skillinvocationresponse_execution_time: (a: number, b: bigint) => void;
  readonly __wbindgen_free: (a: number, b: number, c: number) => void;
  readonly __wbindgen_malloc: (a: number, b: number) => number;
  readonly __wbindgen_realloc: (a: number, b: number, c: number, d: number) => number;
  readonly __wbindgen_export_3: WebAssembly.Table;
  readonly __externref_table_dealloc: (a: number) => void;
  readonly __wbindgen_start: () => void;
}

export type SyncInitInput = BufferSource | WebAssembly.Module;
/**
* Instantiates the given `module`, which can either be bytes or
* a precompiled `WebAssembly.Module`.
*
* @param {{ module: SyncInitInput }} module - Passing `SyncInitInput` directly is deprecated.
*
* @returns {InitOutput}
*/
export function initSync(module: { module: SyncInitInput } | SyncInitInput): InitOutput;

/**
* If `module_or_path` is {RequestInfo} or {URL}, makes a request and
* for everything else, calls `WebAssembly.instantiate` directly.
*
* @param {{ module_or_path: InitInput | Promise<InitInput> }} module_or_path - Passing `InitInput` directly is deprecated.
*
* @returns {Promise<InitOutput>}
*/
export default function __wbg_init (module_or_path?: { module_or_path: InitInput | Promise<InitInput> } | InitInput | Promise<InitInput>): Promise<InitOutput>;
