import { ChangeEvent } from 'react';

import { ChevronDoubleRightIcon } from '@heroicons/react/20/solid';
import classNames from 'classnames';

import Loading from '@components/Loading';
import NetworkSelect from '@components/NetworkSelect';
import useSelection, { Selection } from '@contexts/selection';
import { useUniqueBeaconStateValues, useUniqueExecutionBlockTraceValues } from '@hooks/useQuery';

const tabs: { id: Selection; name: string }[] = [
  { id: Selection.beacon_state, name: 'Beacon state' },
  { id: Selection.execution_block_trace, name: 'Execution block trace' },
  { id: Selection.execution_bad_block, name: 'Execution bad block' },
];

export default function Selector() {
  const { selection: currentSelection, setSelection } = useSelection();

  const {
    data: beaconStateData,
    isLoading: beaconStateIsLoading,
    error: beaconStateError,
  } = useUniqueBeaconStateValues(['network'], currentSelection === Selection.beacon_state);

  const {
    data: executionBlockTraceData,
    isLoading: executionBlockTraceIsLoading,
    error: executionBlockTraceError,
  } = useUniqueExecutionBlockTraceValues(
    ['network'],
    currentSelection === Selection.execution_block_trace,
  );

  const handleTabClick = (selection: Selection) => {
    setSelection(selection);
  };

  let data = undefined;
  let error = undefined;
  let isLoading = true;

  switch (currentSelection) {
    case Selection.beacon_state:
      data = beaconStateData?.network;
      error = beaconStateError;
      isLoading = beaconStateIsLoading;
      break;
    case Selection.execution_block_trace:
      data = executionBlockTraceData?.network;
      error = executionBlockTraceError;
      isLoading = executionBlockTraceIsLoading;
      break;
    case Selection.execution_bad_block:
      break;
  }

  let network = <Loading className="px-4" />;
  if (!isLoading && data && !error) {
    network = <NetworkSelect networks={data} />;
  } else if (error) {
    network = <div className="px-4 text-red-500 font-bold">Error</div>;
  }

  return (
    <div className="bg-white/35">
      <div className="hidden sm:block font-tracoor text-2xl absolute pt-2 pl-2 ">
        <ChevronDoubleRightIcon className="h-10 w-10 text-sky-600/15" aria-hidden="true" />
      </div>
      <div className="sm:hidden">
        <label htmlFor="tabs" className="sr-only">
          Select a tab
        </label>
        <select
          id="tabs"
          name="tabs"
          className="block w-full py-2 pl-3 pr-10 focus:outline-none sm:text-sm bg-white/35"
          value={currentSelection}
          onChange={(e: ChangeEvent<HTMLSelectElement>) => {
            handleTabClick(e.target.value as Selection);
          }}
        >
          {tabs.map((tab) => (
            <option key={tab.id} value={tab.id}>
              {tab.name}
            </option>
          ))}
        </select>
      </div>
      <div className="hidden sm:block">
        <div className="border-b border-gray-200 pl-10">
          <nav className="flex " aria-label="Tabs">
            <div className="grow pt-0.5">
              {tabs.map((tab) => (
                <a
                  key={tab.id}
                  onClick={() => handleTabClick(tab.id)}
                  className={classNames(
                    tab.id === currentSelection
                      ? 'border-sky-500 text-sky-600'
                      : 'border-transparent text-gray-700 hover:border-gray-300 hover:text-gray-600',
                    'whitespace-nowrap border-b-2 px-4 py-4 text-sm font-bold cursor-pointer inline-flex',
                  )}
                  aria-current={tab.id === currentSelection ? 'page' : undefined}
                >
                  {tab.name}
                </a>
              ))}
            </div>
            <div
              className={classNames(
                isLoading || (data && data.length === 1) ? 'hidden' : '',
                'flex-none mr-14 bg-white/35',
              )}
            >
              <div className="flex items-center justify-center pl-4 h-full ">
                <span className="text-sm text-gray-600">Network:</span>
                {network}
              </div>
            </div>
          </nav>
        </div>
      </div>
    </div>
  );
}
