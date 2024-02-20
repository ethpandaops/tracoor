import { XMarkIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';

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

  const [
    beaconStateSlot,
    beaconStateEpoch,
    beaconStateStateRoot,
    beaconStateNode,
    beaconStateNodeImplementation,
    beaconStateNodeVersion,
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

  const clearFilters = () => {
    switch (currentSelection) {
      case Selection.beacon_state:
        setValue('beaconStateSlot', '');
        setValue('beaconStateEpoch', '');
        setValue('beaconStateStateRoot', '');
        setValue('beaconStateNode', '');
        setValue('beaconStateNodeImplementation', '');
        setValue('beaconStateNodeVersion', '');
        break;
      case Selection.execution_block_trace:
        setValue('executionBlockTraceBlockHash', '');
        setValue('executionBlockTraceBlockNumber', '');
        setValue('executionBlockTraceNode', '');
        setValue('executionBlockTraceNodeImplementation', '');
        setValue('executionBlockTraceNodeVersion', '');
        break;
      case Selection.execution_bad_block:
        setValue('executionBadBlockBlockHash', '');
        setValue('executionBadBlockBlockNumber', '');
        setValue('executionBadBlockNode', '');
        setValue('executionBadBlockNodeImplementation', '');
        setValue('executionBadBlockNodeVersion', '');
        break;
    }
  };

  return (
    <div className="bg-white/35 m-10 p-10 rounded-xl">
      <div className="absolute -mt-12 bg-white/65 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold">
        Filters
      </div>
      {hasFilters && (
        <button
          className="absolute right-14 -mt-12 bg-white/85 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-gray-600 font-bold flex cursor-pointer transition hover:text-gray-800"
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
