import BeaconBadBlobInfo from '@components/BeaconBadBlobInfo';
import BeaconBadPermanentStoreBlock from '@components/BeaconBadPermanentStoreBlock';
import BeaconPermanentStoreBlock from '@components/BeaconPermanentStoreBlock';
import BeaconStateInfo from '@components/BeaconStateInfo';
import ExecutionBadPermanentStoreBlock from '@components/ExecutionBadPermanentStoreBlock';
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
      info = <BeaconPermanentStoreBlock />;
      break;
    case Selection.beacon_bad_block:
      info = <BeaconBadPermanentStoreBlock />;
      break;
    case Selection.beacon_bad_blob:
      info = <BeaconBadBlobInfo />;
      break;
    case Selection.execution_block_trace:
      info = <ExecutionBlockTraceInfo />;
      break;
    case Selection.execution_bad_block:
      info = <ExecutionBadPermanentStoreBlock />;
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
