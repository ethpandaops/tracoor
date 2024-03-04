import { useEffect } from 'react';

import { XMarkIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import { useLocation } from 'wouter';

import BeaconBadBlockFilter from '@components/BeaconBadBlockFilter';
import BeaconBlockFilter from '@components/BeaconBlockFilter';
import BeaconStateFilter from '@components/BeaconStateFilter';
import ExecutionBadBlockFilter from '@components/ExecutionBadBlockFilter';
import ExecutionBlockTraceFilter from '@components/ExecutionBlockTraceFilter';
import useSelection, { Selection } from '@contexts/selection';

export interface V1ListUniqueBeaconStateValuesResponse {
  node?: string[];
  slot?: number[];
  epoch?: number[];
  state_root?: string[];
  node_version?: string[];
  network?: string[];
  beacon_implementation?: string[];
}

export default function FilterForm() {
  const { watch, setValue } = useFormContext();
  const { selection: currentSelection } = useSelection();
  const [, navigate] = useLocation();

  const [
    beaconStateSlot,
    beaconStateEpoch,
    beaconStateStateRoot,
    beaconStateNode,
    beaconStateNodeImplementation,
    beaconStateNodeVersion,
    beaconBlockSlot,
    beaconBlockEpoch,
    beaconBlockBlockRoot,
    beaconBlockNode,
    beaconBlockNodeImplementation,
    beaconBlockNodeVersion,
    beaconBadBlockSlot,
    beaconBadBlockEpoch,
    beaconBadBlockBlockRoot,
    beaconBadBlockNode,
    beaconBadBlockNodeImplementation,
    beaconBadBlockNodeVersion,
    executionBlockTraceBlockHash,
    executionBlockTraceBlockNumber,
    executionBlockTraceNode,
    executionBlockTraceNodeImplementation,
    executionBlockTraceNodeVersion,
    executionBadBlockBlockHash,
    executionBadBlockBlockNumber,
    executionBadBlockNode,
    executionBadBlockNodeImplementation,
    executionBadBlockNodeVersion,
  ] = watch([
    'beaconStateSlot',
    'beaconStateEpoch',
    'beaconStateStateRoot',
    'beaconStateNode',
    'beaconStateNodeImplementation',
    'beaconStateNodeVersion',
    'beaconBlockSlot',
    'beaconBlockEpoch',
    'beaconBlockBlockRoot',
    'beaconBlockNode',
    'beaconBlockNodeImplementation',
    'beaconBlockNodeVersion',
    'beaconBadBlockSlot',
    'beaconBadBlockEpoch',
    'beaconBadBlockBlockRoot',
    'beaconBadBlockNode',
    'beaconBadBlockNodeImplementation',
    'beaconBadBlockNodeVersion',
    'executionBlockTraceBlockHash',
    'executionBlockTraceBlockNumber',
    'executionBlockTraceNode',
    'executionBlockTraceNodeImplementation',
    'executionBlockTraceNodeVersion',
    'executionBadBlockBlockHash',
    'executionBadBlockBlockNumber',
    'executionBadBlockNode',
    'executionBadBlockNodeImplementation',
    'executionBadBlockNodeVersion',
  ]);

  let form = undefined;
  let hasFilters = false;
  switch (currentSelection) {
    case Selection.beacon_state:
      form = <BeaconStateFilter />;
      if (
        beaconStateSlot ||
        beaconStateEpoch ||
        beaconStateStateRoot ||
        beaconStateNode ||
        beaconStateNodeImplementation ||
        beaconStateNodeVersion
      ) {
        hasFilters = true;
      }
      break;
    case Selection.beacon_block:
      form = <BeaconBlockFilter />;
      if (
        beaconBlockSlot ||
        beaconBlockEpoch ||
        beaconBlockBlockRoot ||
        beaconBlockNode ||
        beaconBlockNodeImplementation ||
        beaconBlockNodeVersion
      ) {
        hasFilters = true;
      }
      break;
    case Selection.beacon_bad_block:
      form = <BeaconBadBlockFilter />;
      if (
        beaconBadBlockSlot ||
        beaconBadBlockEpoch ||
        beaconBadBlockBlockRoot ||
        beaconBadBlockNode ||
        beaconBadBlockNodeImplementation ||
        beaconBadBlockNodeVersion
      ) {
        hasFilters = true;
      }
      break;
    case Selection.execution_block_trace:
      form = <ExecutionBlockTraceFilter />;
      if (
        executionBlockTraceBlockHash ||
        executionBlockTraceBlockNumber ||
        executionBlockTraceNode ||
        executionBlockTraceNodeImplementation ||
        executionBlockTraceNodeVersion
      ) {
        hasFilters = true;
      }
      break;
    case Selection.execution_bad_block:
      form = <ExecutionBadBlockFilter />;
      if (
        executionBadBlockBlockHash ||
        executionBadBlockBlockNumber ||
        executionBadBlockNode ||
        executionBadBlockNodeImplementation ||
        executionBadBlockNodeVersion
      ) {
        hasFilters = true;
      }
      break;
  }

  useEffect(() => {
    const queryParams = new URLSearchParams();
    switch (currentSelection) {
      case Selection.beacon_state:
        if (beaconStateSlot) queryParams.append('beaconStateSlot', beaconStateSlot);
        if (beaconStateEpoch) queryParams.append('beaconStateEpoch', beaconStateEpoch);
        if (beaconStateStateRoot) queryParams.append('beaconStateStateRoot', beaconStateStateRoot);
        if (beaconStateNode) queryParams.append('beaconStateNode', beaconStateNode);
        if (beaconStateNodeImplementation)
          queryParams.append('beaconStateNodeImplementation', beaconStateNodeImplementation);
        if (beaconStateNodeVersion)
          queryParams.append('beaconStateNodeVersion', beaconStateNodeVersion);
        break;
      case Selection.beacon_block:
        if (beaconBlockSlot) queryParams.append('beaconBlockSlot', beaconBlockSlot);
        if (beaconBlockEpoch) queryParams.append('beaconBlockEpoch', beaconBlockEpoch);
        if (beaconBlockBlockRoot) queryParams.append('beaconBlockBlockRoot', beaconBlockBlockRoot);
        if (beaconBlockNode) queryParams.append('beaconBlockNode', beaconBlockNode);
        if (beaconBlockNodeImplementation)
          queryParams.append('beaconBlockNodeImplementation', beaconBlockNodeImplementation);
        if (beaconBlockNodeVersion)
          queryParams.append('beaconBlockNodeVersion', beaconBlockNodeVersion);
        break;
      case Selection.beacon_bad_block:
        if (beaconBadBlockSlot) queryParams.append('beaconBadBlockSlot', beaconBadBlockSlot);
        if (beaconBadBlockEpoch) queryParams.append('beaconBadBlockEpoch', beaconBadBlockEpoch);
        if (beaconBadBlockBlockRoot)
          queryParams.append('beaconBadBlockBlockRoot', beaconBadBlockBlockRoot);
        if (beaconBadBlockNode) queryParams.append('beaconBadBlockNode', beaconBadBlockNode);
        if (beaconBadBlockNodeImplementation)
          queryParams.append('beaconBadBlockNodeImplementation', beaconBadBlockNodeImplementation);
        if (beaconBadBlockNodeVersion)
          queryParams.append('beaconBadBlockNodeVersion', beaconBadBlockNodeVersion);
        break;
      case Selection.execution_block_trace:
        if (executionBlockTraceBlockHash)
          queryParams.append('executionBlockTraceBlockHash', executionBlockTraceBlockHash);
        if (executionBlockTraceBlockNumber)
          queryParams.append('executionBlockTraceBlockNumber', executionBlockTraceBlockNumber);
        if (executionBlockTraceNode)
          queryParams.append('executionBlockTraceNode', executionBlockTraceNode);
        if (executionBlockTraceNodeImplementation)
          queryParams.append(
            'executionBlockTraceNodeImplementation',
            executionBlockTraceNodeImplementation,
          );
        if (executionBlockTraceNodeVersion)
          queryParams.append('executionBlockTraceNodeVersion', executionBlockTraceNodeVersion);
        break;
      case Selection.execution_bad_block:
        if (executionBadBlockBlockHash)
          queryParams.append('executionBadBlockBlockHash', executionBadBlockBlockHash);
        if (executionBadBlockBlockNumber)
          queryParams.append('executionBadBlockBlockNumber', executionBadBlockBlockNumber);
        if (executionBadBlockNode)
          queryParams.append('executionBadBlockNode', executionBadBlockNode);
        if (executionBadBlockNodeImplementation)
          queryParams.append(
            'executionBadBlockNodeImplementation',
            executionBadBlockNodeImplementation,
          );
        if (executionBadBlockNodeVersion)
          queryParams.append('executionBadBlockNodeVersion', executionBadBlockNodeVersion);
        break;
    }

    const newRelativePathQuery = !hasFilters
      ? window.location.pathname
      : `${window.location.pathname}?${queryParams.toString()}`;

    navigate(newRelativePathQuery, { replace: true });
  }, [
    currentSelection,
    beaconStateSlot,
    beaconStateEpoch,
    beaconStateStateRoot,
    beaconStateNode,
    beaconStateNodeImplementation,
    beaconStateNodeVersion,
    beaconBlockSlot,
    beaconBlockEpoch,
    beaconBlockBlockRoot,
    beaconBlockNode,
    beaconBlockNodeImplementation,
    beaconBlockNodeVersion,
    beaconBadBlockSlot,
    beaconBadBlockEpoch,
    beaconBadBlockBlockRoot,
    beaconBadBlockNode,
    beaconBadBlockNodeImplementation,
    beaconBadBlockNodeVersion,
    executionBlockTraceBlockHash,
    executionBlockTraceBlockNumber,
    executionBlockTraceNode,
    executionBlockTraceNodeImplementation,
    executionBlockTraceNodeVersion,
    executionBadBlockBlockHash,
    executionBadBlockBlockNumber,
    executionBadBlockNode,
    executionBadBlockNodeImplementation,
    executionBadBlockNodeVersion,
    hasFilters,
    navigate,
  ]);

  const clearFilters = () => {
    switch (currentSelection) {
      case Selection.beacon_state:
        setValue('beaconStateSlot', '');
        setValue('beaconStateEpoch', '');
        setValue('beaconStateStateRoot', '');
        setValue('beaconStateNode', null);
        setValue('beaconStateNodeImplementation', null);
        setValue('beaconStateNodeVersion', null);
        break;
      case Selection.beacon_block:
        setValue('beaconBlockSlot', '');
        setValue('beaconBlockEpoch', '');
        setValue('beaconBlockBlockRoot', '');
        setValue('beaconBlockNode', null);
        setValue('beaconBlockNodeImplementation', null);
        setValue('beaconBlockNodeVersion', null);
        break;
      case Selection.beacon_bad_block:
        setValue('beaconBadBlockSlot', '');
        setValue('beaconBadBlockEpoch', '');
        setValue('beaconBadBlockBlockRoot', '');
        setValue('beaconBadBlockNode', null);
        setValue('beaconBadBlockNodeImplementation', null);
        setValue('beaconBadBlockNodeVersion', null);
        break;
      case Selection.execution_block_trace:
        setValue('executionBlockTraceBlockHash', '');
        setValue('executionBlockTraceBlockNumber', '');
        setValue('executionBlockTraceNode', null);
        setValue('executionBlockTraceNodeImplementation', null);
        setValue('executionBlockTraceNodeVersion', null);
        break;
      case Selection.execution_bad_block:
        setValue('executionBadBlockBlockHash', '');
        setValue('executionBadBlockBlockNumber', '');
        setValue('executionBadBlockNode', null);
        setValue('executionBadBlockNodeImplementation', null);
        setValue('executionBadBlockNodeVersion', null);
        break;
    }
  };

  if (!form) return null;

  return (
    <div className="bg-white/35 m-10 p-10 rounded-xl">
      <div className="absolute -mt-12 bg-white/65 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold border-2 border-sky-400">
        Filters
      </div>
      {hasFilters && (
        <button
          className="absolute right-14 -mt-12 bg-white/85 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-gray-600 font-bold flex cursor-pointer transition hover:text-gray-800 border-2 border-gray-500 hover:border-gray-700"
          onClick={clearFilters}
        >
          Clear
          <XMarkIcon className="w-4 h-4" />
        </button>
      )}
      {form}
    </div>
  );
}
