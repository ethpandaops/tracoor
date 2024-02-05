import { ChevronDoubleRightIcon } from '@heroicons/react/20/solid';
import classNames from 'classnames';

import Loading from '@components/Loading';
import NetworkSelect from '@components/NetworkSelect';
import useSelection, { Selection } from '@contexts/selection';
import { useUniqueBeaconStateValues } from '@hooks/useQuery';

const tabs: { id: Selection; name: string }[] = [
  { id: Selection.beacon_state, name: 'Beacon state' },
  { id: Selection.execution_block_trace, name: 'Execution Block trace' },
  { id: Selection.beacon_invalid_blocks, name: 'Invalid gossiped blocks' },
];

export default function Selector() {
  const { selection: currentSelection, setSelection } = useSelection();

  const {
    data: beaconStateData,
    isLoading: beaconStateIsLoading,
    error: beaconStateError,
  } = useUniqueBeaconStateValues(['network'], currentSelection === Selection.beacon_state);

  const handleTabClick = (selection: Selection) => {
    setSelection(selection);
  };

  let data = undefined;
  let error = undefined;
  let isLoading = true;

  console.log('currentSelection', currentSelection);

  switch (currentSelection) {
    case Selection.beacon_state:
      data = beaconStateData?.network;
      error = beaconStateError;
      isLoading = beaconStateIsLoading;
      break;
    case Selection.execution_block_trace:
      break;
    case Selection.beacon_invalid_blocks:
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
          className="block w-full rounded-md border-gray-300 py-2 pl-3 pr-10 text-base focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
          defaultValue={currentSelection}
        >
          {tabs.map((tab) => (
            <option key={tab.id}>{tab.name}</option>
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
            <div className="flex-none mr-14 bg-white/35">
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
