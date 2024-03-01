import { useQuery } from '@tanstack/react-query';

import {
  fetchListUniqueBeaconBadBlockValues,
  fetchListBeaconBadBlock,
  fetchCountBeaconBadBlock,
} from '@api/beaconBadBlock';
import {
  fetchListUniqueBeaconBlockValues,
  fetchListBeaconBlock,
  fetchCountBeaconBlock,
} from '@api/beaconBlock';
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
  BeaconBadBlock,
  BeaconBadBlockField,
  BeaconBlock,
  BeaconBlockField,
  BeaconState,
  BeaconStateField,
  ExecutionBadBlock,
  ExecutionBadBlockField,
  ExecutionBlockTrace,
  ExecutionBlockTraceField,
  V1CountBeaconBadBlockRequest,
  V1CountBeaconBlockRequest,
  V1CountBeaconStateRequest,
  V1CountExecutionBadBlockRequest,
  V1CountExecutionBlockTraceRequest,
  V1ListBeaconBadBlockRequest,
  V1ListBeaconBlockRequest,
  V1ListBeaconStateRequest,
  V1ListExecutionBadBlockRequest,
  V1ListExecutionBlockTraceRequest,
  V1ListUniqueBeaconBadBlockValuesResponse,
  V1ListUniqueBeaconBlockValuesResponse,
  V1ListUniqueBeaconStateValuesResponse,
  V1ListUniqueExecutionBadBlockValuesResponse,
  V1ListUniqueExecutionBlockTraceValuesResponse,
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

export function useBeaconBlocks(request: V1ListBeaconBlockRequest, enabled = true) {
  return useQuery<BeaconBlock[], unknown, BeaconBlock[], [string, V1ListBeaconBlockRequest]>({
    queryKey: ['list-beacon-block', request],
    queryFn: () => fetchListBeaconBlock(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useBeaconBlocksCount(request: V1CountBeaconBlockRequest, enabled = true) {
  return useQuery<number, unknown, number, [string, V1CountBeaconBlockRequest]>({
    queryKey: ['count-beacon-block', request],
    queryFn: () => fetchCountBeaconBlock(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useUniqueBeaconBlockValues(fields: BeaconBlockField[], enabled = true) {
  return useQuery<
    V1ListUniqueBeaconBlockValuesResponse,
    unknown,
    V1ListUniqueBeaconBlockValuesResponse,
    [string, BeaconBlockField[]]
  >({
    queryKey: ['list-unqiue-beacon-block-values', fields],
    queryFn: () => fetchListUniqueBeaconBlockValues({ fields }),
    enabled,
    staleTime: 60_000,
  });
}

export function useBeaconBadBlocks(request: V1ListBeaconBadBlockRequest, enabled = true) {
  return useQuery<
    BeaconBadBlock[],
    unknown,
    BeaconBadBlock[],
    [string, V1ListBeaconBadBlockRequest]
  >({
    queryKey: ['list-beacon-bad-block', request],
    queryFn: () => fetchListBeaconBadBlock(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useBeaconBadBlocksCount(request: V1CountBeaconBadBlockRequest, enabled = true) {
  return useQuery<number, unknown, number, [string, V1CountBeaconBadBlockRequest]>({
    queryKey: ['count-beacon-bad-block', request],
    queryFn: () => fetchCountBeaconBadBlock(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useUniqueBeaconBadBlockValues(fields: BeaconBadBlockField[], enabled = true) {
  return useQuery<
    V1ListUniqueBeaconBadBlockValuesResponse,
    unknown,
    V1ListUniqueBeaconBadBlockValuesResponse,
    [string, BeaconBadBlockField[]]
  >({
    queryKey: ['list-unqiue-beacon-bad-block-values', fields],
    queryFn: () => fetchListUniqueBeaconBadBlockValues({ fields }),
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
