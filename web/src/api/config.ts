import { V1GetConfigResponse, V1GetConfigRequest, Config } from '@app/types/api';
import { BASE_URL } from '@utils/environment';

export async function fetchGetConfig(payload: V1GetConfigRequest): Promise<Config> {
  const response = await fetch(`${BASE_URL}v1/api/get-config`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error('Failed to fetch config data');
  }
  const json = (await response.json()) as V1GetConfigResponse;

  if (json.config === undefined) throw new Error('No config data in response');

  return json.config as Required<Config>;
}
