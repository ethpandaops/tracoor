import {
  V1ListBeaconBadBlockResponse,
  V1ListBeaconBadBlockRequest,
  V1CountBeaconBadBlockResponse,
  V1CountBeaconBadBlockRequest,
  V1ListUniqueBeaconBadBlockValuesResponse,
  V1ListUniqueBeaconBadBlockValuesRequest,
  BeaconBadBlock,
} from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchListBeaconBadBlock(
  payload: V1ListBeaconBadBlockRequest,
): Promise<BeaconBadBlock[]> {
  const response = await fetch(`${BASE_URL}v1/api/list-beacon-bad-block`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon bad block data');
  }
  const json = (await response.json()) as V1ListBeaconBadBlockResponse;

  if (json.beacon_bad_blocks === undefined)
    throw new Error('No beacon bad blocks data in response');

  return json.beacon_bad_blocks as Required<BeaconBadBlock[]>;
}

export async function fetchCountBeaconBadBlock(
  payload: V1CountBeaconBadBlockRequest,
): Promise<number> {
  const response = await fetch(`${BASE_URL}v1/api/count-beacon-bad-block`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch count beacon bad block data');
  }
  const json = (await response.json()) as V1CountBeaconBadBlockResponse;

  return json.count ?? 0;
}

export async function fetchListUniqueBeaconBadBlockValues(
  payload: V1ListUniqueBeaconBadBlockValuesRequest,
): Promise<V1ListUniqueBeaconBadBlockValuesResponse> {
  const response = await fetch(`${BASE_URL}v1/api/list-unique-beacon-bad-block-values`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon bad block data');
  }
  const beaconBlocks = (await response.json()) as V1ListUniqueBeaconBadBlockValuesResponse;

  if (beaconBlocks === undefined)
    throw new Error('No unique beacon bad blocks values data in response');

  return beaconBlocks as Required<V1ListUniqueBeaconBadBlockValuesResponse>;
}
