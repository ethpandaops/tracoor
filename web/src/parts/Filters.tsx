import BeaconStateFilter from '@components/BeaconStateFilter';
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
  const { selection: currentSelection } = useSelection();

  let form = undefined;
  switch (currentSelection) {
    case Selection.beacon_state:
      form = <BeaconStateFilter />;
      break;
    case Selection.execution_block_trace:
      form = <ExecutionBlockTraceFilter />;
      break;
    case Selection.beacon_invalid_blocks:
      break;
  }

  return (
    <div className="bg-white/35 m-10 p-10 rounded-xl">
      <div className="absolute -mt-12 bg-white/65 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold">
        Filters
      </div>
      {form}
    </div>
  );
}
