import {
  V1ListBeaconBadBlobResponse,
  V1ListBeaconBadBlobRequest,
  V1CountBeaconBadBlobResponse,
  V1CountBeaconBadBlobRequest,
  V1ListUniqueBeaconBadBlobValuesResponse,
  V1ListUniqueBeaconBadBlobValuesRequest,
  BeaconBadBlob,
} from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchListBeaconBadBlob(
  payload: V1ListBeaconBadBlobRequest,
): Promise<BeaconBadBlob[]> {
  const response = await fetch(`${BASE_URL}v1/api/list-beacon-bad-blob`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon bad blob data');
  }
  const json = (await response.json()) as V1ListBeaconBadBlobResponse;

  if (json.beacon_bad_blobs === undefined) throw new Error('No beacon bad blobs data in response');

  return json.beacon_bad_blobs as Required<BeaconBadBlob[]>;
}

export async function fetchCountBeaconBadBlob(
  payload: V1CountBeaconBadBlobRequest,
): Promise<number> {
  const response = await fetch(`${BASE_URL}v1/api/count-beacon-bad-blob`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch count beacon bad blob data');
  }
  const json = (await response.json()) as V1CountBeaconBadBlobResponse;

  return json.count ?? 0;
}

export async function fetchListUniqueBeaconBadBlobValues(
  payload: V1ListUniqueBeaconBadBlobValuesRequest,
): Promise<V1ListUniqueBeaconBadBlobValuesResponse> {
  const response = await fetch(`${BASE_URL}v1/api/list-unique-beacon-bad-blob-values`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon bad blob data');
  }
  const beaconBlobs = (await response.json()) as V1ListUniqueBeaconBadBlobValuesResponse;

  if (beaconBlobs === undefined)
    throw new Error('No unique beacon bad blobs values data in response');

  return beaconBlobs as Required<V1ListUniqueBeaconBadBlobValuesResponse>;
}
