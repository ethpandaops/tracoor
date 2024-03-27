import React, { ReactNode } from 'react';

import { useForm, FormProvider } from 'react-hook-form';

export interface Props {
  children: ReactNode;
  beaconStateSlot?: string;
  beaconStateEpoch?: string;
  beaconStateStateRoot?: string;
  beaconStateNode?: string | null;
  beaconStateNodeImplementation?: string | null;
  beaconStateNodeVersion?: string | null;
  executionBlockTraceBlockHash?: string;
  executionBlockTraceBlockNumber?: string;
  executionBlockTraceNode?: string | null;
  executionBlockTraceNodeImplementation?: string | null;
  executionBlockTraceNodeVersion?: string | null;
  executionBadBlockBlockHash?: string;
  executionBadBlockBlockNumber?: string;
  executionBadBlockNode?: string | null;
  executionBadBlockNodeImplementation?: string | null;
  executionBadBlockNodeVersion?: string | null;
  goEvmLabDiffBlockHash?: string;
  goEvmLabDiffBlockNumber?: string;
  goEvmLabDiffTx?: string;
  executionBlockTraceSelectorId?: string;
  executionBlockTraceSelectorBlockHash?: string;
  executionBlockTraceSelectorBlockNumber?: string;
  executionBlockTraceSelectorId1?: string;
  executionBlockTraceSelectorBlockHash1?: string;
  executionBlockTraceSelectorBlockNumber1?: string;
  executionBlockTraceSelectorId2?: string;
  executionBlockTraceSelectorBlockHash2?: string;
  executionBlockTraceSelectorBlockNumber2?: string;
  beaconBlockSelectorId?: string;
  beaconBlockSelectorSlot?: string;
  beaconBlockSelectorBlockRoot?: string;
  beaconStateSelectorId?: string;
  beaconStateSelectorSlot?: string;
  beaconStateSelectorStateRoot?: string;
  zcliFileName?: string;
}

export default function Provider({ children }: Props) {
  const queryParams = new URLSearchParams(window.location.search);

  const methods = useForm<Props>({
    defaultValues: {
      beaconStateSlot: queryParams.get('beaconStateSlot') || '',
      beaconStateEpoch: queryParams.get('beaconStateEpoch') || '',
      beaconStateStateRoot: queryParams.get('beaconStateStateRoot') || '',
      beaconStateNode: queryParams.get('beaconStateNode') || null,
      beaconStateNodeImplementation: queryParams.get('beaconStateNodeImplementation') || null,
      beaconStateNodeVersion: queryParams.get('beaconStateNodeVersion') || null,
      executionBlockTraceBlockHash: queryParams.get('executionBlockTraceBlockHash') || '',
      executionBlockTraceBlockNumber: queryParams.get('executionBlockTraceBlockNumber') || '',
      executionBlockTraceNode: queryParams.get('executionBlockTraceNode') || null,
      executionBlockTraceNodeImplementation:
        queryParams.get('executionBlockTraceNodeImplementation') || null,
      executionBlockTraceNodeVersion: queryParams.get('executionBlockTraceNodeVersion') || null,
      executionBadBlockBlockHash: queryParams.get('executionBadBlockBlockHash') || '',
      executionBadBlockBlockNumber: queryParams.get('executionBadBlockBlockNumber') || '',
      executionBadBlockNode: queryParams.get('executionBadBlockNode') || null,
      executionBadBlockNodeImplementation:
        queryParams.get('executionBadBlockNodeImplementation') || null,
      executionBadBlockNodeVersion: queryParams.get('executionBadBlockNodeVersion') || null,
      goEvmLabDiffBlockHash: queryParams.get('goEvmLabDiffBlockHash') || '',
      goEvmLabDiffBlockNumber: queryParams.get('goEvmLabDiffBlockNumber') || '',
      goEvmLabDiffTx: queryParams.get('goEvmLabDiffTx') || '',
      executionBlockTraceSelectorId: queryParams.get('executionBlockTraceSelectorId') || '',
      executionBlockTraceSelectorBlockHash:
        queryParams.get('executionBlockTraceSelectorBlockHash') || '',
      executionBlockTraceSelectorBlockNumber:
        queryParams.get('executionBlockTraceSelectorBlockNumber') || '',
      executionBlockTraceSelectorId1: queryParams.get('executionBlockTraceSelectorId1') || '',
      executionBlockTraceSelectorBlockHash1:
        queryParams.get('executionBlockTraceSelectorBlockHash1') || '',
      executionBlockTraceSelectorBlockNumber1:
        queryParams.get('executionBlockTraceSelectorBlockNumber1') || '',
      executionBlockTraceSelectorId2: queryParams.get('executionBlockTraceSelectorId2') || '',
      executionBlockTraceSelectorBlockHash2:
        queryParams.get('executionBlockTraceSelectorBlockHash2') || '',
      executionBlockTraceSelectorBlockNumber2:
        queryParams.get('executionBlockTraceSelectorBlockNumber2') || '',
      beaconBlockSelectorId: queryParams.get('beaconBlockSelectorId') || '',
      beaconBlockSelectorSlot: queryParams.get('beaconBlockSelectorSlot') || '',
      beaconBlockSelectorBlockRoot: queryParams.get('beaconBlockSelectorBlockRoot') || '',
      beaconStateSelectorId: queryParams.get('beaconStateSelectorId') || '',
      beaconStateSelectorSlot: queryParams.get('beaconStateSelectorSlot') || '',
      beaconStateSelectorStateRoot: queryParams.get('beaconStateSelectorStateRoot') || '',
      zcliFileName: queryParams.get('zcliFileName') || '',
    },
  });
  return <FormProvider {...methods}>{children}</FormProvider>;
}
