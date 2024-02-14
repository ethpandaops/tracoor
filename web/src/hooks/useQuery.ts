import { useQuery } from '@tanstack/react-query';

import {
  fetchListUniqueBeaconStateValues,
  fetchListBeaconState,
  fetchCountBeaconState,
} from '@api/beaconState';
import {
  fetchListUniqueExecutionBadBlockValues,
  fetchListExecutionBadBlock,
  fetchCountExecutionBadBlock,
} from '@api/executionBadBlock';
import {
  fetchListUniqueExecutionBlockTraceValues,
  fetchListExecutionBlockTrace,
  fetchCountExecutionBlockTrace,
} from '@api/executionBlockTrace';
import {
  BeaconStateField,
  V1ListUniqueBeaconStateValuesResponse,
  BeaconState,
  V1ListBeaconStateRequest,
  V1CountBeaconStateRequest,
  ExecutionBlockTraceField,
  V1ListUniqueExecutionBlockTraceValuesResponse,
  ExecutionBlockTrace,
  V1ListExecutionBlockTraceRequest,
  V1CountExecutionBlockTraceRequest,
  ExecutionBadBlockField,
  V1ListUniqueExecutionBadBlockValuesResponse,
  ExecutionBadBlock,
  V1ListExecutionBadBlockRequest,
  V1CountExecutionBadBlockRequest,
} from '@app/types/api';

export function useBeaconStates(request: V1ListBeaconStateRequest, enabled = true) {
  return useQuery<BeaconState[], unknown, BeaconState[], [string, V1ListBeaconStateRequest]>({
    queryKey: ['list-beacon-state', request],
    queryFn: () => fetchListBeaconState(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useBeaconStatesCount(request: V1CountBeaconStateRequest, enabled = true) {
  return useQuery<number, unknown, number, [string, V1CountBeaconStateRequest]>({
    queryKey: ['count-beacon-state', request],
    queryFn: () => fetchCountBeaconState(request),
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

export function useExecutionBlockTracesCount(
  request: V1CountExecutionBlockTraceRequest,
  enabled = true,
) {
  return useQuery<number, unknown, number, [string, V1CountExecutionBlockTraceRequest]>({
    queryKey: ['count-execution-block-trace', request],
    queryFn: () => fetchCountExecutionBlockTrace(request),
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

export function useExecutionBadBlocks(request: V1ListExecutionBadBlockRequest, enabled = true) {
  return useQuery<
    ExecutionBadBlock[],
    unknown,
    ExecutionBadBlock[],
    [string, V1ListExecutionBadBlockRequest]
  >({
    queryKey: ['list-execution-bad-block', request],
    queryFn: () => fetchListExecutionBadBlock(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useExecutionBadBlocksCount(
  request: V1CountExecutionBadBlockRequest,
  enabled = true,
) {
  return useQuery<number, unknown, number, [string, V1CountExecutionBadBlockRequest]>({
    queryKey: ['count-execution-bad-block', request],
    queryFn: () => fetchCountExecutionBadBlock(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useUniqueExecutionBadBlockValues(fields: ExecutionBadBlockField[], enabled = true) {
  return useQuery<
    V1ListUniqueExecutionBadBlockValuesResponse,
    unknown,
    V1ListUniqueExecutionBadBlockValuesResponse,
    [string, ExecutionBadBlockField[]]
  >({
    queryKey: ['list-unqiue-execution-bad-block-values', fields],
    queryFn: () => fetchListUniqueExecutionBadBlockValues({ fields }),
    enabled,
    staleTime: 60_000,
  });
}
