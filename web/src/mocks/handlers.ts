import { http, RequestHandler, HttpResponse } from 'msw';

import { V1ListBeaconStateResponse, V1ListUniqueBeaconStateValuesResponse } from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export const beaconStates: Required<V1ListBeaconStateResponse> = {
  beacon_states: [
    {
      id: 123,
      node: 'test-node',
      fetched_at: new Date().toISOString(),
      slot: 1,
      epoch: 1,
      state_root: '0x1',
      node_version: 'node-version-1',
      network: 'test-network',
      beacon_implementation: 'test-beacon-implementation',
    },
  ],
};

export const beaconStatesUniqueValues: Required<V1ListUniqueBeaconStateValuesResponse> = {
  node: ['test-node'],
  slot: [1],
  epoch: [1],
  state_root: ['0x1'],
  node_version: ['node-version-1'],
  network: ['test-network'],
  beacon_implementation: ['test-beacon-implementation'],
};

export const handlers: Array<RequestHandler> = [
  http.post(`${BASE_URL}v1/api/list-beacon-state`, () => {
    return HttpResponse.json(beaconStates);
  }),
  http.post(`${BASE_URL}v1/api/list-unique-beacon-state-values`, () => {
    return HttpResponse.json(beaconStatesUniqueValues);
  }),
];
