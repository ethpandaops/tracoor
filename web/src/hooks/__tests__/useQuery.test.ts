import { renderHook, waitFor } from '@testing-library/react';

import { beaconStatesUniqueValues, beaconStates } from '@app/mocks/handlers';
import { useBeaconStates, useUniqueBeaconStateValues } from '@hooks/useQuery';
import { ProviderWrapper } from '@utils/testing';

describe('useQuery', () => {
  describe('useBeaconStates', () => {
    it('should return metadata', async () => {
      const { result } = renderHook(() => useBeaconStates({ node: 'test' }), {
        wrapper: ProviderWrapper(),
      });

      await waitFor(() => result.current.isSuccess);
      await waitFor(() => expect(result.current).toEqual(beaconStates.beacon_states));
    });
  });

  describe('useUniqueBeaconStateValues', () => {
    it('should return current slot and epoch', async () => {
      const { result } = renderHook(() => useUniqueBeaconStateValues([]), {
        wrapper: ProviderWrapper(),
      });

      await waitFor(() => result.current.isSuccess);
      await waitFor(() => expect(result.current).toEqual(beaconStatesUniqueValues));
    });
  });
});
