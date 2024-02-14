import BeaconStateTable from '@components/BeaconStateTable';
import ExecutionBadBlockTable from '@components/ExecutionBadBlockTable';
import ExecutionBlockTraceTable from '@components/ExecutionBlockTraceTable';
import Loading from '@components/Loading';
import useNetwork from '@contexts/network';
import useSelection, { Selection } from '@contexts/selection';

export default function Listing() {
  const { selection: currentSelection } = useSelection();

  const { network } = useNetwork();

  let table = undefined;
  switch (currentSelection) {
    case Selection.beacon_state:
      table = <BeaconStateTable />;
      break;
    case Selection.execution_block_trace:
      table = <ExecutionBlockTraceTable />;
      break;
    case Selection.execution_bad_block:
      table = <ExecutionBadBlockTable />;
      break;
  }

  if (!network) {
    table = <Loading message="Waiting for network selection" />;
  }

  return <div className="px-4 sm:px-6 lg:px-8">{table}</div>;
}
