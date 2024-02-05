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
}

export default function Provider({ children }: Props) {
  const methods = useForm<Props>({
    defaultValues: {
      beaconStateNode: null,
      beaconStateNodeImplementation: null,
      beaconStateNodeVersion: null,
    },
  });
  return <FormProvider {...methods}>{children}</FormProvider>;
}
