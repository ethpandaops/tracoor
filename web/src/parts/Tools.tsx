import { useEffect } from 'react';

import { useFormContext } from 'react-hook-form';
import { useLocation } from 'wouter';

import GoEVMLabDiff from '@app/components/GoEVMLabDiff';
import LCLIStateTransition from '@components/LCLIStateTransition';
import NCLIStateTransition from '@components/NCLIStateTransition';
import ZCLIDiff from '@components/ZCLIDiff';
import useSelection, { Selection } from '@contexts/selection';

export default function Tools() {
  const { watch } = useFormContext();
  const [, navigate] = useLocation();
  const { selection: currentSelection } = useSelection();

  const [
    beaconBlockSelectorId,
    beaconBlockSelectorSlot,
    beaconBlockSelectorBlockRoot,
    beaconStateSelectorId,
    beaconStateSelectorSlot,
    beaconStateSelectorStateRoot,
    goEvmLabDiffTx,
    executionBlockTraceSelectorId1,
    executionBlockTraceSelectorBlockHash1,
    executionBlockTraceSelectorBlockNumber1,
    executionBlockTraceSelectorId2,
    executionBlockTraceSelectorBlockHash2,
    executionBlockTraceSelectorBlockNumber2,
    zcliFileName,
  ] = watch([
    'beaconBlockSelectorId',
    'beaconBlockSelectorSlot',
    'beaconBlockSelectorBlockRoot',
    'beaconStateSelectorId',
    'beaconStateSelectorSlot',
    'beaconStateSelectorStateRoot',
    'goEvmLabDiffTx',
    'executionBlockTraceSelectorId1',
    'executionBlockTraceSelectorBlockHash1',
    'executionBlockTraceSelectorBlockNumber1',
    'executionBlockTraceSelectorId2',
    'executionBlockTraceSelectorBlockHash2',
    'executionBlockTraceSelectorBlockNumber2',
    'zcliFileName',
  ]);

  useEffect(() => {
    const queryParams = new URLSearchParams();
    switch (currentSelection) {
      case Selection.zcli_state_diff:
        if (zcliFileName) queryParams.append('zcliFileName', zcliFileName);
        if (beaconStateSelectorId)
          queryParams.append('beaconStateSelectorId', beaconStateSelectorId);
        if (beaconStateSelectorSlot)
          queryParams.append('beaconStateSelectorSlot', beaconStateSelectorSlot);
        if (beaconStateSelectorStateRoot)
          queryParams.append('beaconStateSelectorStateRoot', beaconStateSelectorStateRoot);
        break;
      case Selection.ncli_state_transition:
      case Selection.lcli_state_transition:
        if (beaconBlockSelectorId)
          queryParams.append('beaconBlockSelectorId', beaconBlockSelectorId);
        if (beaconBlockSelectorSlot)
          queryParams.append('beaconBlockSelectorSlot', beaconBlockSelectorSlot);
        if (beaconBlockSelectorBlockRoot)
          queryParams.append('beaconBlockSelectorBlockRoot', beaconBlockSelectorBlockRoot);
        if (beaconStateSelectorId)
          queryParams.append('beaconStateSelectorId', beaconStateSelectorId);
        if (beaconStateSelectorSlot)
          queryParams.append('beaconStateSelectorSlot', beaconStateSelectorSlot);
        if (beaconStateSelectorStateRoot)
          queryParams.append('beaconStateSelectorStateRoot', beaconStateSelectorStateRoot);
        break;
      case Selection.go_evm_lab_diff:
        if (goEvmLabDiffTx) queryParams.append('goEvmLabDiffTx', goEvmLabDiffTx);
        if (executionBlockTraceSelectorId1)
          queryParams.append('executionBlockTraceSelectorId1', executionBlockTraceSelectorId1);
        if (executionBlockTraceSelectorBlockHash1)
          queryParams.append(
            'executionBlockTraceSelectorBlockHash1',
            executionBlockTraceSelectorBlockHash1,
          );
        if (executionBlockTraceSelectorBlockNumber1)
          queryParams.append(
            'executionBlockTraceSelectorBlockNumber1',
            executionBlockTraceSelectorBlockNumber1,
          );
        if (executionBlockTraceSelectorId2)
          queryParams.append('executionBlockTraceSelectorId2', executionBlockTraceSelectorId2);
        if (executionBlockTraceSelectorBlockHash2)
          queryParams.append(
            'executionBlockTraceSelectorBlockHash2',
            executionBlockTraceSelectorBlockHash2,
          );
        if (executionBlockTraceSelectorBlockNumber2)
          queryParams.append(
            'executionBlockTraceSelectorBlockNumber2',
            executionBlockTraceSelectorBlockNumber2,
          );
        break;
    }

    const newRelativePathQuery = !queryParams.toString()
      ? window.location.pathname
      : `${window.location.pathname}?${queryParams.toString()}`;

    navigate(newRelativePathQuery, { replace: true });
  }, [
    currentSelection,
    beaconBlockSelectorId,
    beaconBlockSelectorSlot,
    beaconBlockSelectorBlockRoot,
    beaconStateSelectorId,
    beaconStateSelectorSlot,
    beaconStateSelectorStateRoot,
    goEvmLabDiffTx,
    executionBlockTraceSelectorId1,
    executionBlockTraceSelectorBlockHash1,
    executionBlockTraceSelectorBlockNumber1,
    executionBlockTraceSelectorId2,
    executionBlockTraceSelectorBlockHash2,
    executionBlockTraceSelectorBlockNumber2,
    zcliFileName,
    navigate,
  ]);

  let comp = undefined;
  switch (currentSelection) {
    case Selection.go_evm_lab_diff:
      comp = <GoEVMLabDiff />;
      break;
    case Selection.lcli_state_transition:
      comp = <LCLIStateTransition />;
      break;
    case Selection.ncli_state_transition:
      comp = <NCLIStateTransition />;
      break;
    case Selection.zcli_state_diff:
      comp = <ZCLIDiff />;
      break;
    default:
      return null;
  }

  return <div className="px-4 sm:px-6 lg:px-8">{comp}</div>;
}
