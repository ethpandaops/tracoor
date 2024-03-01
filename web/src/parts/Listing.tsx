import BeaconBadBlockTable from '@components/BeaconBadBlockTable';
import BeaconBlockTable from '@components/BeaconBlockTable';
import BeaconStateTable from '@components/BeaconStateTable';
import ExecutionBadBlockTable from '@components/ExecutionBadBlockTable';
import ExecutionBlockTraceTable from '@components/ExecutionBlockTraceTable';
import useSelection, { Selection } from '@contexts/selection';

export default function Listing({ id }: { id?: string }) {
  const { selection: currentSelection } = useSelection();

  let table = undefined;
  switch (currentSelection) {
    case Selection.beacon_state:
      table = <BeaconStateTable id={id} />;
      break;
    case Selection.beacon_block:
      table = <BeaconBlockTable id={id} />;
      break;
    case Selection.beacon_bad_block:
      table = <BeaconBadBlockTable id={id} />;
      break;
    case Selection.execution_block_trace:
      table = <ExecutionBlockTraceTable id={id} />;
      break;
    case Selection.execution_bad_block:
      table = <ExecutionBadBlockTable id={id} />;
      break;
    default:
      return null;
  }

  return <div className="px-4 sm:px-6 lg:px-8">{table}</div>;
}
