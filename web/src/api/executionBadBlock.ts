import {
  V1ListExecutionBadBlockResponse,
  V1ListExecutionBadBlockRequest,
  V1CountExecutionBadBlockResponse,
  V1CountExecutionBadBlockRequest,
  V1ListUniqueExecutionBadBlockValuesResponse,
  V1ListUniqueExecutionBadBlockValuesRequest,
  ExecutionBadBlock,
} from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchListExecutionBadBlock(
  payload: V1ListExecutionBadBlockRequest,
): Promise<ExecutionBadBlock[]> {
  const response = await fetch(`${BASE_URL}v1/api/list-execution-bad-block`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list execution bad block data');
  }
  const json = (await response.json()) as V1ListExecutionBadBlockResponse;

  if (json.execution_bad_blocks === undefined)
    throw new Error('No execution bad block data in response');

  return json.execution_bad_blocks as Required<ExecutionBadBlock[]>;
}

export async function fetchCountExecutionBadBlock(
  payload: V1CountExecutionBadBlockRequest,
): Promise<number> {
  const response = await fetch(`${BASE_URL}v1/api/count-execution-bad-block`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch count execution bad block data');
  }
  const json = (await response.json()) as V1CountExecutionBadBlockResponse;

  return json.count ?? 0;
}

export async function fetchListUniqueExecutionBadBlockValues(
  payload: V1ListUniqueExecutionBadBlockValuesRequest,
): Promise<V1ListUniqueExecutionBadBlockValuesResponse> {
  const response = await fetch(`${BASE_URL}v1/api/list-unique-execution-bad-block-values`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list execution bad block data');
  }
  const beaconStates = (await response.json()) as V1ListUniqueExecutionBadBlockValuesResponse;

  if (beaconStates === undefined)
    throw new Error('No unique execution bad block values data in response');

  return beaconStates as Required<V1ListUniqueExecutionBadBlockValuesResponse>;
}
