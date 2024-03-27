import { ChangeEvent, useEffect, useState, useMemo } from 'react';

import { Tab } from '@headlessui/react';
import { ChevronDoubleRightIcon } from '@heroicons/react/20/solid';
import classNames from 'classnames';
import { useLocation, Link } from 'wouter';

import Loading from '@components/Loading';
import NetworkSelect from '@components/NetworkSelect';
import useSelection, { Selection } from '@contexts/selection';
import {
  useUniqueBeaconStateValues,
  useUniqueBeaconBlockValues,
  useUniqueBeaconBadBlockValues,
  useUniqueExecutionBlockTraceValues,
  useUniqueExecutionBadBlockValues,
} from '@hooks/useQuery';

const categories: { name: string; tabs: Selection[] }[] = [
  {
    name: 'Consensus Layer',
    tabs: [Selection.beacon_state, Selection.beacon_block, Selection.beacon_bad_block],
  },
  {
    name: 'Execution Layer',
    tabs: [Selection.execution_block_trace, Selection.execution_bad_block],
  },
  {
    name: 'Tools',
    tabs: [
      Selection.go_evm_lab_diff,
      Selection.lcli_state_transition,
      Selection.ncli_state_transition,
      Selection.zcli_state_diff,
    ],
  },
];

const tabs: { id: Selection; name: string }[] = [
  { id: Selection.beacon_state, name: 'Beacon states' },
  { id: Selection.beacon_block, name: 'Beacon blocks' },
  { id: Selection.beacon_bad_block, name: 'Beacon bad blocks' },
  { id: Selection.execution_block_trace, name: 'Execution block traces' },
  { id: Selection.execution_bad_block, name: 'Execution bad blocks' },
  { id: Selection.go_evm_lab_diff, name: 'Go EVM lab diff' },
  { id: Selection.lcli_state_transition, name: 'lcli state transition' },
  { id: Selection.ncli_state_transition, name: 'ncli state transition' },
  { id: Selection.zcli_state_diff, name: 'zcli state diff' },
];

export default function Selector() {
  const { selection: currentSelection, setSelection } = useSelection();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [location, setLocation] = useLocation();

  useEffect(() => {
    const index = categories.findIndex((category) => category.tabs.includes(currentSelection));
    if (index !== -1 && index !== selectedIndex) setSelectedIndex(index);
  }, [currentSelection]);

  const locationCategory = useMemo(() => {
    return categories.find((category) => category.tabs.includes(currentSelection));
  }, [currentSelection]);

  const {
    data: beaconStateData,
    isLoading: beaconStateIsLoading,
    error: beaconStateError,
  } = useUniqueBeaconStateValues(
    ['network'],
    [
      Selection.beacon_state,
      Selection.ncli_state_transition,
      Selection.lcli_state_transition,
      Selection.zcli_state_diff,
    ].includes(currentSelection),
  );

  const {
    data: beaconBlockData,
    isLoading: beaconBlockIsLoading,
    error: beaconBlockError,
  } = useUniqueBeaconBlockValues(['network'], currentSelection === Selection.beacon_block);

  const {
    data: beaconBadBlockData,
    isLoading: beaconBadBlockIsLoading,
    error: beaconBadBlockError,
  } = useUniqueBeaconBadBlockValues(['network'], currentSelection === Selection.beacon_bad_block);

  const {
    data: executionBlockTraceData,
    isLoading: executionBlockTraceIsLoading,
    error: executionBlockTraceError,
  } = useUniqueExecutionBlockTraceValues(
    ['network'],
    [Selection.execution_block_trace, Selection.go_evm_lab_diff].includes(currentSelection),
  );

  const {
    data: executionBadBlockData,
    isLoading: executionBadBlockIsLoading,
    error: executionBadBlockError,
  } = useUniqueExecutionBadBlockValues(
    ['network'],
    currentSelection === Selection.execution_bad_block,
  );

  const handleTabClick = (selection: Selection) => {
    setLocation(`/${selection}`);
    setSelection(selection);
  };

  useEffect(() => {
    const path = location.split('/')[1] as Selection;
    switch (path) {
      case Selection.beacon_state:
        if (currentSelection !== Selection.beacon_state) setSelection(Selection.beacon_state);
        break;
      case Selection.beacon_block:
        if (currentSelection !== Selection.beacon_block) setSelection(Selection.beacon_block);
        break;
      case Selection.beacon_bad_block:
        if (currentSelection !== Selection.beacon_bad_block)
          setSelection(Selection.beacon_bad_block);
        break;
      case Selection.execution_block_trace:
        if (currentSelection !== Selection.execution_block_trace)
          setSelection(Selection.execution_block_trace);
        break;
      case Selection.execution_bad_block:
        if (currentSelection !== Selection.execution_bad_block)
          setSelection(Selection.execution_bad_block);
        break;
      case Selection.go_evm_lab_diff:
        if (currentSelection !== Selection.go_evm_lab_diff) setSelection(Selection.go_evm_lab_diff);
        break;
      case Selection.lcli_state_transition:
        if (currentSelection !== Selection.lcli_state_transition)
          setSelection(Selection.lcli_state_transition);
        break;
      case Selection.ncli_state_transition:
        if (currentSelection !== Selection.ncli_state_transition)
          setSelection(Selection.ncli_state_transition);
        break;
      case Selection.zcli_state_diff:
        if (currentSelection !== Selection.zcli_state_diff) setSelection(Selection.zcli_state_diff);
        break;
      default:
        if (currentSelection !== Selection.beacon_state) setSelection(Selection.beacon_state);
        break;
    }
  }, [location]);

  let data = undefined;
  let error = undefined;
  let isLoading = true;

  switch (currentSelection) {
    case Selection.beacon_state:
    case Selection.ncli_state_transition:
    case Selection.lcli_state_transition:
    case Selection.zcli_state_diff:
      data = beaconStateData?.network;
      error = beaconStateError;
      isLoading = beaconStateIsLoading;
      break;
    case Selection.beacon_block:
      data = beaconBlockData?.network;
      error = beaconBlockError;
      isLoading = beaconBlockIsLoading;
      break;
    case Selection.beacon_bad_block:
      data = beaconBadBlockData?.network;
      error = beaconBadBlockError;
      isLoading = beaconBadBlockIsLoading;
      break;
    case Selection.execution_block_trace:
    case Selection.go_evm_lab_diff:
      data = executionBlockTraceData?.network;
      error = executionBlockTraceError;
      isLoading = executionBlockTraceIsLoading;
      break;
    case Selection.execution_bad_block:
      data = executionBadBlockData?.network;
      error = executionBadBlockError;
      isLoading = executionBadBlockIsLoading;
      break;
    default:
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
      <div className="sm:hidden font-tracoor text-2xl absolute pt-[8px] left-1">
        <ChevronDoubleRightIcon className="h-6 w-6 text-sky-600/35" aria-hidden="true" />
      </div>
      <div className="sm:hidden">
        <label htmlFor="tabs" className="sr-only">
          Select a tab
        </label>
        <select
          id="tabs"
          name="tabs"
          className="block w-full py-2.5 pl-8 font-bold text-sky-600 pr-10 focus:outline-none sm:text-sm bg-white/35"
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
        <Tab.Group selectedIndex={selectedIndex} onChange={setSelectedIndex}>
          <Tab.List>
            <div className="border-b border-gray-200 pl-10">
              <div className="flex">
                <div className="grow pt-0.5">
                  {categories.map((category) => (
                    <Tab
                      key={category.name}
                      className={({ selected }) =>
                        classNames(
                          selected
                            ? 'border-sky-500 text-sky-600'
                            : 'border-transparent text-gray-700 hover:border-b-gray-300 hover:text-gray-600',
                          'whitespace-nowrap border-b-2 px-4 py-4 text-sm font-bold cursor-pointer inline-flex',
                          locationCategory?.name === category.name
                            ? 'text-sky-600 hover:text-sky-700'
                            : '',
                        )
                      }
                    >
                      {category.name}
                    </Tab>
                  ))}
                </div>
              </div>
            </div>
          </Tab.List>
          <Tab.Panels>
            <div className="border-b border-gray-200 pl-10 bg-white/35">
              <div className="flex">
                <div className="grow pt-0.5">
                  {categories.map((category) => (
                    <Tab.Panel key={category.name}>
                      <ul>
                        {category.tabs.map((selection) => {
                          const tab = tabs.find((tab) => tab.id === selection);
                          if (!tab) return null;
                          return (
                            <Link
                              key={tab.id}
                              href={`/${tab.id}`}
                              className={classNames(
                                tab.id === currentSelection
                                  ? 'border-sky-500 text-sky-600'
                                  : 'border-transparent text-gray-700 hover:border-b-gray-300 hover:text-gray-600',
                                'whitespace-nowrap border-b-2 px-4 py-2 text-sm font-bold cursor-pointer inline-flex',
                                locationCategory?.name === category.name
                                  ? 'border-b-2 border-sky-500'
                                  : '',
                              )}
                              aria-current={tab.id === currentSelection ? 'page' : undefined}
                            >
                              {tab.name}
                            </Link>
                          );
                        })}
                      </ul>
                    </Tab.Panel>
                  ))}
                </div>
              </div>
            </div>
          </Tab.Panels>
        </Tab.Group>
      </div>
      <div className="hidden">{network}</div>
    </div>
  );
}
