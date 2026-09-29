import { useQuery } from '@tanstack/react-query';

import {
  fetchListUniqueBeaconBadBlobValues,
  fetchListBeaconBadBlob,
  fetchCountBeaconBadBlob,
} from '@api/beaconBadBlob';
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
import { fetchGetConfig } from '@api/config';
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
  fetchListUniqueExecutionPayloadEnvelopeValues,
  fetchListExecutionPayloadEnvelope,
  fetchCountExecutionPayloadEnvelope,
} from '@api/executionPayloadEnvelope';
import {
  BeaconBadBlock,
  BeaconBadBlob,
  BeaconBadBlockField,
  BeaconBadBlobField,
  BeaconBlock,
  BeaconBlockField,
  BeaconState,
  BeaconStateField,
  ExecutionBadBlock,
  ExecutionBadBlockField,
  ExecutionBlockTrace,
  ExecutionBlockTraceField,
  ExecutionPayloadEnvelope,
  ExecutionPayloadEnvelopeField,
  V1CountBeaconBadBlockRequest,
  V1CountBeaconBadBlobRequest,
  V1CountBeaconBlockRequest,
  V1CountBeaconStateRequest,
  V1CountExecutionBadBlockRequest,
  V1CountExecutionBlockTraceRequest,
  V1CountExecutionPayloadEnvelopeRequest,
  V1ListBeaconBadBlockRequest,
  V1ListBeaconBadBlobRequest,
  V1ListBeaconBlockRequest,
  V1ListBeaconStateRequest,
  V1ListExecutionBadBlockRequest,
  V1ListExecutionBlockTraceRequest,
  V1ListExecutionPayloadEnvelopeRequest,
  V1ListUniqueBeaconBadBlockValuesResponse,
  V1ListUniqueBeaconBadBlobValuesResponse,
  V1ListUniqueBeaconBlockValuesResponse,
  V1ListUniqueBeaconStateValuesResponse,
  V1ListUniqueExecutionBadBlockValuesResponse,
  V1ListUniqueExecutionBlockTraceValuesResponse,
  V1ListUniqueExecutionPayloadEnvelopeValuesResponse,
  V1GetConfigRequest,
  Config,
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

export function useUniqueBeaconStateValues(
  fields: BeaconStateField[],
  network?: string,
  enabled = true,
) {
  return useQuery<
    V1ListUniqueBeaconStateValuesResponse,
    unknown,
    V1ListUniqueBeaconStateValuesResponse,
    [string, BeaconStateField[], string | undefined]
  >({
    queryKey: ['list-unique-beacon-state-values', fields, network],
    queryFn: () => fetchListUniqueBeaconStateValues({ fields, network }),
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

export function useUniqueBeaconBlockValues(
  fields: BeaconBlockField[],
  network?: string,
  enabled = true,
) {
  return useQuery<
    V1ListUniqueBeaconBlockValuesResponse,
    unknown,
    V1ListUniqueBeaconBlockValuesResponse,
    [string, BeaconBlockField[], string | undefined]
  >({
    queryKey: ['list-unique-beacon-block-values', fields, network],
    queryFn: () => fetchListUniqueBeaconBlockValues({ fields, network }),
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

export function useUniqueBeaconBadBlockValues(
  fields: BeaconBadBlockField[],
  network?: string,
  enabled = true,
) {
  return useQuery<
    V1ListUniqueBeaconBadBlockValuesResponse,
    unknown,
    V1ListUniqueBeaconBadBlockValuesResponse,
    [string, BeaconBadBlockField[], string | undefined]
  >({
    queryKey: ['list-unique-beacon-bad-block-values', fields, network],
    queryFn: () => fetchListUniqueBeaconBadBlockValues({ fields, network }),
    enabled,
    staleTime: 60_000,
  });
}

export function useBeaconBadBlobs(request: V1ListBeaconBadBlobRequest, enabled = true) {
  return useQuery<BeaconBadBlob[], unknown, BeaconBadBlob[], [string, V1ListBeaconBadBlobRequest]>({
    queryKey: ['list-beacon-bad-blob', request],
    queryFn: () => fetchListBeaconBadBlob(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useBeaconBadBlobsCount(request: V1CountBeaconBadBlobRequest, enabled = true) {
  return useQuery<number, unknown, number, [string, V1CountBeaconBadBlobRequest]>({
    queryKey: ['count-beacon-bad-blob', request],
    queryFn: () => fetchCountBeaconBadBlob(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useUniqueBeaconBadBlobValues(
  fields: BeaconBadBlobField[],
  network?: string,
  enabled = true,
) {
  return useQuery<
    V1ListUniqueBeaconBadBlobValuesResponse,
    unknown,
    V1ListUniqueBeaconBadBlobValuesResponse,
    [string, BeaconBadBlobField[], string | undefined]
  >({
    queryKey: ['list-unique-beacon-bad-blob-values', fields, network],
    queryFn: () => fetchListUniqueBeaconBadBlobValues({ fields, network }),
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
  network?: string,
  enabled = true,
) {
  return useQuery<
    V1ListUniqueExecutionBlockTraceValuesResponse,
    unknown,
    V1ListUniqueExecutionBlockTraceValuesResponse,
    [string, ExecutionBlockTraceField[], string | undefined]
  >({
    queryKey: ['list-unique-execution-block-trace-values', fields, network],
    queryFn: () => fetchListUniqueExecutionBlockTraceValues({ fields, network }),
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

export function useUniqueExecutionBadBlockValues(
  fields: ExecutionBadBlockField[],
  network?: string,
  enabled = true,
) {
  return useQuery<
    V1ListUniqueExecutionBadBlockValuesResponse,
    unknown,
    V1ListUniqueExecutionBadBlockValuesResponse,
    [string, ExecutionBadBlockField[], string | undefined]
  >({
    queryKey: ['list-unique-execution-bad-block-values', fields, network],
    queryFn: () => fetchListUniqueExecutionBadBlockValues({ fields, network }),
    enabled,
    staleTime: 60_000,
  });
}

export function useExecutionPayloadEnvelopes(
  request: V1ListExecutionPayloadEnvelopeRequest,
  enabled = true,
) {
  return useQuery<
    ExecutionPayloadEnvelope[],
    unknown,
    ExecutionPayloadEnvelope[],
    [string, V1ListExecutionPayloadEnvelopeRequest]
  >({
    queryKey: ['list-execution-payload-envelope', request],
    queryFn: () => fetchListExecutionPayloadEnvelope(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useExecutionPayloadEnvelopesCount(
  request: V1CountExecutionPayloadEnvelopeRequest,
  enabled = true,
) {
  return useQuery<number, unknown, number, [string, V1CountExecutionPayloadEnvelopeRequest]>({
    queryKey: ['count-execution-payload-envelope', request],
    queryFn: () => fetchCountExecutionPayloadEnvelope(request),
    enabled,
    staleTime: 6_000,
  });
}

export function useUniqueExecutionPayloadEnvelopeValues(
  fields: ExecutionPayloadEnvelopeField[],
  network?: string,
  enabled = true,
) {
  return useQuery<
    V1ListUniqueExecutionPayloadEnvelopeValuesResponse,
    unknown,
    V1ListUniqueExecutionPayloadEnvelopeValuesResponse,
    [string, ExecutionPayloadEnvelopeField[], string | undefined]
  >({
    queryKey: ['list-unique-execution-payload-envelope-values', fields, network],
    queryFn: () => fetchListUniqueExecutionPayloadEnvelopeValues({ fields, network }),
    enabled,
    staleTime: 60_000,
  });
}

export function useConfig(request: V1GetConfigRequest, enabled = true) {
  return useQuery<Config, unknown, Config, [string, V1GetConfigRequest]>({
    queryKey: ['get-config', request],
    queryFn: () => fetchGetConfig(request),
    enabled,
    staleTime: 600_000,
  });
}
