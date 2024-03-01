import {
  V1ListBeaconBlockResponse,
  V1ListBeaconBlockRequest,
  V1CountBeaconBlockResponse,
  V1CountBeaconBlockRequest,
  V1ListUniqueBeaconBlockValuesResponse,
  V1ListUniqueBeaconBlockValuesRequest,
  BeaconBlock,
} from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchListBeaconBlock(
  payload: V1ListBeaconBlockRequest,
): Promise<BeaconBlock[]> {
  const response = await fetch(`${BASE_URL}v1/api/list-beacon-block`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon block data');
  }
  const json = (await response.json()) as V1ListBeaconBlockResponse;

  if (json.beacon_blocks === undefined) throw new Error('No beacon blocks data in response');

  return json.beacon_blocks as Required<BeaconBlock[]>;
}

export async function fetchCountBeaconBlock(payload: V1CountBeaconBlockRequest): Promise<number> {
  const response = await fetch(`${BASE_URL}v1/api/count-beacon-block`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch count beacon block data');
  }
  const json = (await response.json()) as V1CountBeaconBlockResponse;

  return json.count ?? 0;
}

export async function fetchListUniqueBeaconBlockValues(
  payload: V1ListUniqueBeaconBlockValuesRequest,
): Promise<V1ListUniqueBeaconBlockValuesResponse> {
  const response = await fetch(`${BASE_URL}v1/api/list-unique-beacon-block-values`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon block data');
  }
  const beaconBlocks = (await response.json()) as V1ListUniqueBeaconBlockValuesResponse;

  if (beaconBlocks === undefined)
    throw new Error('No unique beacon blocks values data in response');

  return beaconBlocks as Required<V1ListUniqueBeaconBlockValuesResponse>;
}
