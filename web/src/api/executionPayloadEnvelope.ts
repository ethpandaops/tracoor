import {
  V1ListExecutionPayloadEnvelopeResponse,
  V1ListExecutionPayloadEnvelopeRequest,
  V1CountExecutionPayloadEnvelopeResponse,
  V1CountExecutionPayloadEnvelopeRequest,
  V1ListUniqueExecutionPayloadEnvelopeValuesResponse,
  V1ListUniqueExecutionPayloadEnvelopeValuesRequest,
  ExecutionPayloadEnvelope,
} from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchListExecutionPayloadEnvelope(
  payload: V1ListExecutionPayloadEnvelopeRequest,
): Promise<ExecutionPayloadEnvelope[]> {
  const response = await fetch(`${BASE_URL}v1/api/list-execution-payload-envelope`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list execution payload envelope data');
  }
  const json = (await response.json()) as V1ListExecutionPayloadEnvelopeResponse;

  if (json.execution_payload_envelopes === undefined)
    throw new Error('No execution payload envelopes data in response');

  return json.execution_payload_envelopes as Required<ExecutionPayloadEnvelope[]>;
}

export async function fetchCountExecutionPayloadEnvelope(
  payload: V1CountExecutionPayloadEnvelopeRequest,
): Promise<number> {
  const response = await fetch(`${BASE_URL}v1/api/count-execution-payload-envelope`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch count execution payload envelope data');
  }
  const json = (await response.json()) as V1CountExecutionPayloadEnvelopeResponse;

  return json.count ?? 0;
}

export async function fetchListUniqueExecutionPayloadEnvelopeValues(
  payload: V1ListUniqueExecutionPayloadEnvelopeValuesRequest,
): Promise<V1ListUniqueExecutionPayloadEnvelopeValuesResponse> {
  const response = await fetch(`${BASE_URL}v1/api/list-unique-execution-payload-envelope-values`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch list execution payload envelope data');
  }
  const executionPayloadEnvelopes =
    (await response.json()) as V1ListUniqueExecutionPayloadEnvelopeValuesResponse;

  if (executionPayloadEnvelopes === undefined)
    throw new Error('No unique execution payload envelopes values data in response');

  return executionPayloadEnvelopes as Required<V1ListUniqueExecutionPayloadEnvelopeValuesResponse>;
}
