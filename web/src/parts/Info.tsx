import BeaconBadBlobInfo from '@components/BeaconBadBlobInfo';
import BeaconBadBlockInfo from '@components/BeaconBadBlockInfo';
import BeaconBlockInfo from '@components/BeaconBlockInfo';
import BeaconStateInfo from '@components/BeaconStateInfo';
import ExecutionBadBlockInfo from '@components/ExecutionBadBlockInfo';
import ExecutionBlockTraceInfo from '@components/ExecutionBlockTraceInfo';
import GoEVMLabInfo from '@components/GoEVMLabInfo';
import LCLIInfo from '@components/LCLIInfo';
import NCLIInfo from '@components/NCLIInfo';
import ZCLIInfo from '@components/ZCLIInfo';
import useSelection, { Selection } from '@contexts/selection';

export default function Info() {
  const { selection: currentSelection } = useSelection();

  let info = undefined;
  switch (currentSelection) {
    case Selection.beacon_state:
      info = <BeaconStateInfo />;
      break;
    case Selection.beacon_block:
      info = <BeaconBlockInfo />;
      break;
    case Selection.beacon_bad_block:
      info = <BeaconBadBlockInfo />;
      break;
    case Selection.beacon_bad_blob:
      info = <BeaconBadBlobInfo />;
      break;
    case Selection.execution_block_trace:
      info = <ExecutionBlockTraceInfo />;
      break;
    case Selection.execution_bad_block:
      info = <ExecutionBadBlockInfo />;
      break;
    case Selection.ncli_state_transition:
      info = <NCLIInfo />;
      break;
    case Selection.lcli_state_transition:
      info = <LCLIInfo />;
      break;
    case Selection.go_evm_lab_diff:
      info = <GoEVMLabInfo />;
      break;
    case Selection.zcli_state_diff:
      info = <ZCLIInfo />;
      break;
    default:
      return null;
  }

  return <div className="px-4 sm:px-6 lg:px-8">{info}</div>;
}
