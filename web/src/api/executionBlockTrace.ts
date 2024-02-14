import {
  V1ListExecutionBlockTraceResponse,
  V1ListExecutionBlockTraceRequest,
  V1CountExecutionBlockTraceResponse,
  V1CountExecutionBlockTraceRequest,
  V1ListUniqueExecutionBlockTraceValuesResponse,
  V1ListUniqueExecutionBlockTraceValuesRequest,
  ExecutionBlockTrace,
} from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchListExecutionBlockTrace(
  payload: V1ListExecutionBlockTraceRequest,
): Promise<ExecutionBlockTrace[]> {
  const response = await fetch(`${BASE_URL}v1/api/list-execution-block-trace`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list execution block trace data');
  }
  const json = (await response.json()) as V1ListExecutionBlockTraceResponse;

  if (json.execution_block_traces === undefined)
    throw new Error('No execution block trace data in response');

  return json.execution_block_traces as Required<ExecutionBlockTrace[]>;
}

export async function fetchCountExecutionBlockTrace(
  payload: V1CountExecutionBlockTraceRequest,
): Promise<number> {
  const response = await fetch(`${BASE_URL}v1/api/count-execution-block-trace`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch count execution block trace data');
  }
  const json = (await response.json()) as V1CountExecutionBlockTraceResponse;

  return json.count ?? 0;
}

export async function fetchListUniqueExecutionBlockTraceValues(
  payload: V1ListUniqueExecutionBlockTraceValuesRequest,
): Promise<V1ListUniqueExecutionBlockTraceValuesResponse> {
  const response = await fetch(`${BASE_URL}v1/api/list-unique-execution-block-trace-values`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list execution block trace data');
  }
  const beaconStates = (await response.json()) as V1ListUniqueExecutionBlockTraceValuesResponse;

  if (beaconStates === undefined)
    throw new Error('No unique execution block trace values data in response');

  return beaconStates as Required<V1ListUniqueExecutionBlockTraceValuesResponse>;
}
