import { Config } from '@app/types/api';

export function isCustomNetwork(config: Config) {
  return (
    config?.ethereum?.config?.repository &&
    config?.ethereum?.config?.branch &&
    config?.ethereum?.config?.path
  );
}

export function getNetworkConfig(config: Config): {
  repository: string;
  branch: string;
  path: string;
} {
  return {
    repository: config?.ethereum?.config?.repository ?? '',
    branch: config?.ethereum?.config?.branch ?? '',
    path: config?.ethereum?.config?.path ?? '',
  };
}

export function getNCLIConfig(config: Config): {
  repository: string;
  branch: string;
} {
  return {
    repository: config?.ethereum?.tools?.ncli?.repository
      ? config?.ethereum?.tools?.ncli?.repository
      : 'gstatus-im/nimbus-eth2',
    branch: config?.ethereum?.tools?.ncli?.branch ? config?.ethereum?.tools?.ncli?.branch : '',
  };
}

export function getLCLIConfig(config: Config): {
  repository: string;
  branch: string;
} {
  return {
    repository: config?.ethereum?.tools?.lcli?.repository
      ? config?.ethereum?.tools?.lcli?.repository
      : 'sigp/lighthouse',
    branch: config?.ethereum?.tools?.lcli?.branch ? config?.ethereum?.tools?.lcli?.branch : '',
  };
}

// The fork zcli is told to parse a BeaconState as, when the server config does not name
// one. A stale guess produces a command that simply fails to parse the state, so this
// tracks the newest fork tracoor decodes rather than sitting on whatever was current when
// the fallback was written. Deployments on an older fork should set
// server.ethereum.tools.zcli.fork explicitly.
const DEFAULT_ZCLI_FORK = 'gloas';

export function getZCLIConfig(config: Config): {
  fork: string;
} {
  return {
    fork: config?.ethereum?.tools?.zcli?.fork
      ? config?.ethereum?.tools?.zcli?.fork
      : DEFAULT_ZCLI_FORK,
  };
}
