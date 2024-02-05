import { useEffect, useCallback, useRef } from 'react';

import { useFormContext } from 'react-hook-form';

import BeaconStateTable from '@components/BeaconStateTable';
import ExecutionBlockTraceTable from '@components/ExecutionBlockTraceTable';
import useSelection, { Selection } from '@contexts/selection';

export default function Listing() {
  const { selection: currentSelection } = useSelection();

  let table = undefined;
  switch (currentSelection) {
    case Selection.beacon_state:
      table = <BeaconStateTable />;
      break;
    case Selection.execution_block_trace:
      table = <ExecutionBlockTraceTable />;
      break;
    case Selection.beacon_invalid_blocks:
      break;
  }

  return (
    <div className="px-4 sm:px-6 lg:px-8">
      <div className="mt-8 flow-root">
        <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
          <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">{table}</div>
        </div>
      </div>
    </div>
  );
}
