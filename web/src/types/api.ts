export interface BeaconState {
  id: number;
  node: string;
  fetched_at: string;
  slot: number;
  epoch: number;
  state_root: string;
  node_version: string;
  network: string;
  beacon_implementation: string;
}

export interface ExecutionBlockTrace {
  id: number;
  node: string;
  fetched_at: string;
  block_hash: string;
  block_number: number;
  node_version: string;
  network: string;
  execution_implementation: string;
}

export type BeaconStateField =
  | 'node'
  | 'slot'
  | 'epoch'
  | 'state_root'
  | 'node_version'
  | 'network'
  | 'beacon_implementation';

export type ExecutionBlockTraceField =
  | 'node'
  | 'block_hash'
  | 'block_number'
  | 'node_version'
  | 'network'
  | 'execution_implementation';

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
  id?: number;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueBeaconStateValuesRequest {
  fields: BeaconStateField[];
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
  id?: number;
  pagination?: PaginationCursor;
}

export interface V1ListUniqueExecutionBlockTraceValuesRequest {
  fields: ExecutionBlockTraceField[];
}

/* RESPONSE */
export interface V1ListBeaconStateResponse {
  beacon_states?: BeaconState[];
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

export interface V1ListExecutionBlockTraceResponse {
  execution_block_traces?: ExecutionBlockTrace[];
}

export interface V1ListUniqueExecutionBlockTraceValuesResponse {
  node?: string[];
  block_hash?: string[];
  block_number?: number[];
  node_version?: string[];
  network?: string[];
  execution_implementation?: string[];
}
