import { useQuery } from '@tanstack/react-query';

import { fetchListUniqueBeaconStateValues, fetchListBeaconState } from '@api/beaconState';
import {
  fetchListUniqueExecutionBlockTraceValues,
  fetchListExecutionBlockTrace,
} from '@api/executionBlockTrace';
import {
  BeaconStateField,
  V1ListUniqueBeaconStateValuesResponse,
  BeaconState,
  V1ListBeaconStateRequest,
  ExecutionBlockTraceField,
  V1ListUniqueExecutionBlockTraceValuesResponse,
  ExecutionBlockTrace,
  V1ListExecutionBlockTraceRequest,
} from '@app/types/api';

export function useBeaconStates(request: V1ListBeaconStateRequest, enabled = true) {
  return useQuery<BeaconState[], unknown, BeaconState[], [string, V1ListBeaconStateRequest]>({
    queryKey: ['list-beacon-state', request],
    queryFn: () => fetchListBeaconState(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useUniqueBeaconStateValues(fields: BeaconStateField[], enabled = true) {
  return useQuery<
    V1ListUniqueBeaconStateValuesResponse,
    unknown,
    V1ListUniqueBeaconStateValuesResponse,
    [string, BeaconStateField[]]
  >({
    queryKey: ['list-unqiue-beacon-state-values', fields],
    queryFn: () => fetchListUniqueBeaconStateValues({ fields }),
    enabled,
    staleTime: 60_000,
  });
}

export function useExecutionBlockTraces(request: V1ListExecutionBlockTraceRequest, enabled = true) {
  return useQuery<
    ExecutionBlockTrace[],
    unknown,
    ExecutionBlockTrace[],
    [string, V1ListExecutionBlockTraceRequest]
  >({
    queryKey: ['list-execution-block-trace', request],
    queryFn: () => fetchListExecutionBlockTrace(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useUniqueExecutionBlockTraceValues(
  fields: ExecutionBlockTraceField[],
  enabled = true,
) {
  return useQuery<
    V1ListUniqueExecutionBlockTraceValuesResponse,
    unknown,
    V1ListUniqueExecutionBlockTraceValuesResponse,
    [string, ExecutionBlockTraceField[]]
  >({
    queryKey: ['list-unqiue-execution-block-trace-values', fields],
    queryFn: () => fetchListUniqueExecutionBlockTraceValues({ fields }),
    enabled,
    staleTime: 60_000,
  });
}
