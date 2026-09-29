export interface BeaconState {
  id: string;
  node: string;
  fetched_at: string;
  slot: number;
  epoch: number;
  state_root: string;
  node_version: string;
  network: string;
  beacon_implementation: string;
  content_hash?: string;
  verified_at?: string;
  content_matched_at?: string;
  agreement_count?: number;
}

export interface BeaconBlock {
  id: string;
  node: string;
  fetched_at: string;
  slot: number;
  epoch: number;
  block_root: string;
  node_version: string;
  network: string;
  beacon_implementation: string;
  content_hash?: string;
  verified_at?: string;
  content_matched_at?: string;
  agreement_count?: number;
}

export interface ExecutionPayloadEnvelope {
  id: string;
  node: string;
  fetched_at: string;
  slot: number;
  epoch: number;
  block_root: string;
  node_version: string;
  network: string;
  beacon_implementation: string;
  content_hash?: string;
  verified_at?: string;
  content_matched_at?: string;
  agreement_count?: number;
}

export interface BeaconBadBlock {
  id: string;
  node: string;
  fetched_at: string;
  slot: number;
  epoch: number;
  block_root: string;
  node_version: string;
  network: string;
  beacon_implementation: string;
  content_hash?: string;
  verified_at?: string;
  content_matched_at?: string;
}

export interface BeaconBadBlob {
  id: string;
  node: string;
  fetched_at: string;
  slot: number;
  epoch: number;
  block_root: string;
  node_version: string;
  network: string;
  beacon_implementation: string;
  index: number;
  content_hash?: string;
  verified_at?: string;
  content_matched_at?: string;
}

export interface ExecutionBlockTrace {
  id: string;
  node: string;
  fetched_at: string;
  block_hash: string;
  block_number: number;
  node_version: string;
  network: string;
  execution_implementation: string;
  content_hash?: string;
  verified_at?: string;
  content_matched_at?: string;
  agreement_count?: number;
}

export interface ExecutionBadBlock {
  id: string;
  node: string;
  fetched_at: string;
  block_hash: string;
  block_number: number;
  node_version: string;
  network: string;
  execution_implementation: string;
  block_extra_data: string;
  content_hash?: string;
  verified_at?: string;
  content_matched_at?: string;
}

export type BeaconStateField =
  | 'node'
  | 'slot'
  | 'epoch'
  | 'state_root'
  | 'node_version'
  | 'network'
  | 'beacon_implementation';

export type BeaconBlockField =
  | 'node'
  | 'slot'
  | 'epoch'
  | 'block_root'
  | 'node_version'
  | 'network'
  | 'beacon_implementation';

export type ExecutionPayloadEnvelopeField =
  | 'node'
  | 'slot'
  | 'epoch'
  | 'block_root'
  | 'node_version'
  | 'network'
  | 'beacon_implementation';

export type BeaconBadBlockField =
  | 'node'
  | 'slot'
  | 'epoch'
  | 'block_root'
  | 'node_version'
  | 'network'
  | 'beacon_implementation';

export type BeaconBadBlobField =
  | 'node'
  | 'slot'
  | 'epoch'
  | 'block_root'
  | 'node_version'
  | 'network'
  | 'beacon_implementation'
  | 'index';

export type ExecutionBlockTraceField =
  | 'node'
  | 'block_hash'
  | 'block_number'
  | 'node_version'
  | 'network'
  | 'execution_implementation';

export type ExecutionBadBlockField =
  | 'node'
  | 'block_hash'
  | 'block_number'
  | 'node_version'
  | 'network'
  | 'execution_implementation'
  | 'block_extra_data';

export interface Config {
  ethereum?: {
    config?: {
      repository?: string;
      branch?: string;
      path?: string;
    };
    tools?: {
      ncli?: {
        repository?: string;
        branch?: string;
      };
      lcli?: {
        repository?: string;
        branch?: string;
      };
      zcli?: {
        fork?: string;
      };
    };
  };
}

/* REQUESTS */
export interface PaginationCursor {
  limit?: number;
  offset?: number;
  order_by?: string;
}

export interface V1ListBeaconStateRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  state_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1CountBeaconStateRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  state_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueBeaconStateValuesRequest {
  fields: BeaconStateField[];
  /** Empty means unscoped; only the network dropdown itself leaves it unset. */
  network?: string;
}

export interface V1ListBeaconBlockRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1CountBeaconBlockRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueBeaconBlockValuesRequest {
  fields: BeaconBlockField[];
  /** Empty means unscoped; only the network dropdown itself leaves it unset. */
  network?: string;
}

export interface V1ListExecutionPayloadEnvelopeRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1CountExecutionPayloadEnvelopeRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueExecutionPayloadEnvelopeValuesRequest {
  fields: ExecutionPayloadEnvelopeField[];
  /** Empty means unscoped; only the network dropdown itself leaves it unset. */
  network?: string;
}

export interface V1ListBeaconBadBlockRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1CountBeaconBadBlockRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueBeaconBadBlockValuesRequest {
  fields: BeaconBadBlockField[];
  /** Empty means unscoped; only the network dropdown itself leaves it unset. */
  network?: string;
}

export interface V1ListBeaconBadBlobRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  index?: number;
  pagination?: PaginationCursor;
}

export interface V1CountBeaconBadBlobRequest {
  node?: string;
  slot?: number;
  epoch?: number;
  block_root?: string;
  node_version?: string;
  network?: string;
  beacon_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  index?: number;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueBeaconBadBlobValuesRequest {
  fields: BeaconBadBlobField[];
  /** Empty means unscoped; only the network dropdown itself leaves it unset. */
  network?: string;
}

export interface V1ListExecutionBlockTraceRequest {
  node?: string;
  block_number?: number;
  block_hash?: string;
  node_version?: string;
  network?: string;
  execution_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1CountExecutionBlockTraceRequest {
  node?: string;
  block_number?: number;
  block_hash?: string;
  node_version?: string;
  network?: string;
  execution_implementation?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueExecutionBlockTraceValuesRequest {
  fields: ExecutionBlockTraceField[];
  /** Empty means unscoped; only the network dropdown itself leaves it unset. */
  network?: string;
}

export interface V1ListExecutionBadBlockRequest {
  node?: string;
  block_number?: number;
  block_hash?: string;
  node_version?: string;
  network?: string;
  execution_implementation?: string;
  block_extra_data?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1CountExecutionBadBlockRequest {
  node?: string;
  block_number?: number;
  block_hash?: string;
  node_version?: string;
  network?: string;
  execution_implementation?: string;
  block_extra_data?: string;
  before?: string;
  after?: string;
  id?: string;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueExecutionBadBlockValuesRequest {
  fields: ExecutionBadBlockField[];
  /** Empty means unscoped; only the network dropdown itself leaves it unset. */
  network?: string;
}

export interface V1GetConfigRequest {}

/* RESPONSE */
export interface V1ListBeaconStateResponse {
  beacon_states?: BeaconState[];
}

export interface V1CountBeaconStateResponse {
  count?: number;
}

export interface V1ListUniqueBeaconStateValuesResponse {
  node?: string[];
  slot?: number[];
  epoch?: number[];
  state_root?: string[];
  node_version?: string[];
  network?: string[];
  beacon_implementation?: string[];
}

export interface V1ListBeaconBlockResponse {
  beacon_blocks?: BeaconBlock[];
}

export interface V1CountBeaconBlockResponse {
  count?: number;
}

export interface V1ListUniqueBeaconBlockValuesResponse {
  node?: string[];
  slot?: number[];
  epoch?: number[];
  block_root?: string[];
  node_version?: string[];
  network?: string[];
  beacon_implementation?: string[];
}

export interface V1ListExecutionPayloadEnvelopeResponse {
  execution_payload_envelopes?: ExecutionPayloadEnvelope[];
}

export interface V1CountExecutionPayloadEnvelopeResponse {
  count?: number;
}

export interface V1ListUniqueExecutionPayloadEnvelopeValuesResponse {
  node?: string[];
  slot?: number[];
  epoch?: number[];
  block_root?: string[];
  node_version?: string[];
  network?: string[];
  beacon_implementation?: string[];
}

export interface V1ListBeaconBadBlockResponse {
  beacon_bad_blocks?: BeaconBadBlock[];
}

export interface V1CountBeaconBadBlockResponse {
  count?: number;
}

export interface V1ListUniqueBeaconBadBlockValuesResponse {
  node?: string[];
  slot?: number[];
  epoch?: number[];
  block_root?: string[];
  node_version?: string[];
  network?: string[];
  beacon_implementation?: string[];
}

export interface V1ListBeaconBadBlobResponse {
  beacon_bad_blobs?: BeaconBadBlob[];
}

export interface V1CountBeaconBadBlobResponse {
  count?: number;
}

export interface V1ListUniqueBeaconBadBlobValuesResponse {
  node?: string[];
  slot?: number[];
  epoch?: number[];
  block_root?: string[];
  node_version?: string[];
  network?: string[];
  beacon_implementation?: string[];
  index?: number[];
}

export interface V1ListExecutionBlockTraceResponse {
  execution_block_traces?: ExecutionBlockTrace[];
}

export interface V1CountExecutionBlockTraceResponse {
  count?: number;
}

export interface V1ListUniqueExecutionBlockTraceValuesResponse {
  node?: string[];
  block_hash?: string[];
  block_number?: number[];
  node_version?: string[];
  network?: string[];
  execution_implementation?: string[];
}

export interface V1ListExecutionBadBlockResponse {
  execution_bad_blocks?: ExecutionBadBlock[];
}

export interface V1CountExecutionBadBlockResponse {
  count?: number;
}

export interface V1ListUniqueExecutionBadBlockValuesResponse {
  node?: string[];
  block_hash?: string[];
  block_number?: number[];
  node_version?: string[];
  network?: string[];
  execution_implementation?: string[];
  block_extra_data?: string[];
}

export interface V1GetConfigResponse {
  config: Config;
}
