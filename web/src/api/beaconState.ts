import {
  V1ListBeaconStateResponse,
  V1ListBeaconStateRequest,
  V1CountBeaconStateResponse,
  V1CountBeaconStateRequest,
  V1ListUniqueBeaconStateValuesResponse,
  V1ListUniqueBeaconStateValuesRequest,
  BeaconState,
} from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchListBeaconState(
  payload: V1ListBeaconStateRequest,
): Promise<BeaconState[]> {
  const response = await fetch(`${BASE_URL}v1/api/list-beacon-state`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon state data');
  }
  const json = (await response.json()) as V1ListBeaconStateResponse;

  if (json.beacon_states === undefined) throw new Error('No beacon states data in response');

  return json.beacon_states as Required<BeaconState[]>;
}

export async function fetchCountBeaconState(payload: V1CountBeaconStateRequest): Promise<number> {
  const response = await fetch(`${BASE_URL}v1/api/count-beacon-state`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch count beacon state data');
  }
  const json = (await response.json()) as V1CountBeaconStateResponse;

  return json.count ?? 0;
}

export async function fetchListUniqueBeaconStateValues(
  payload: V1ListUniqueBeaconStateValuesRequest,
): Promise<V1ListUniqueBeaconStateValuesResponse> {
  const response = await fetch(`${BASE_URL}v1/api/list-unique-beacon-state-values`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list beacon state data');
  }
  const beaconStates = (await response.json()) as V1ListUniqueBeaconStateValuesResponse;

  if (beaconStates === undefined)
    throw new Error('No unique beacon states values data in response');

  return beaconStates as Required<V1ListUniqueBeaconStateValuesResponse>;
}
