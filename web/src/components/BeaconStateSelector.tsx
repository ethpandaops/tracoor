import { useMemo } from 'react';

import { XMarkIcon } from '@heroicons/react/24/outline';
import { useFormContext, Controller } from 'react-hook-form';
import TimeAgo from 'react-timeago';

import Alert from '@components/Alert';
import DebouncedInput from '@components/DebouncedInput';
import Loading from '@components/Loading';
import useNetwork from '@contexts/network';
import { useBeaconStates } from '@hooks/useQuery';

export default function BeaconStateSelector() {
  const { control, watch, setValue } = useFormContext();
  const { network } = useNetwork();

  const [beaconStateSelectorId, beaconStateSelectorSlot, beaconStateSelectorStateRoot] = watch([
    'beaconStateSelectorId',
    'beaconStateSelectorSlot',
    'beaconStateSelectorStateRoot',
  ]);

  const {
    data: selectedData,
    isLoading: selectedIsLoading,
    error: selectedError,
  } = useBeaconStates(
    {
      network: network ? network : undefined,
      id: beaconStateSelectorId,
      pagination: {
        limit: 1,
      },
    },
    Boolean(beaconStateSelectorId),
  );

  const state = selectedData?.[0];

  const {
    data: searchData,
    isLoading: searchIsLoading,
    error: searchError,
  } = useBeaconStates(
    {
      network: network ? network : undefined,
      slot: beaconStateSelectorSlot ? beaconStateSelectorSlot : undefined,
      state_root: beaconStateSelectorStateRoot ? beaconStateSelectorStateRoot : undefined,
    },
    Boolean(!beaconStateSelectorId && (beaconStateSelectorSlot || beaconStateSelectorStateRoot)),
  );

  let otherComp = undefined;

  if (selectedIsLoading) {
    otherComp = (
      <dl className="grid grid-cols-1 sm:grid-cols-3 mt-5">
        <div className="px-4 sm:col-span-1 sm:px-0 pb-4 sm:pt-4">
          <dt className="text-sm font-bold leading-6 text-gray-700">Node</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-64 sm:w-32 xl:w-64 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="border-t sm:border-none border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Slot</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-20 sm:w-32 xl:w-24 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="border-t sm:border-none border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Epoch</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-16 sm:w-32 xl:w-20 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Beacon node Implementation</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-32 xl:w-28 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Node version</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-64 sm:w-32 xl:w-64 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">State root</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 font-mono truncate h-5 w-64 sm:w-32 2xl:w-64 3xl:w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
      </dl>
    );
  } else if (selectedError) {
    let message = 'Something went wrong fetching data';
    if (typeof selectedError === 'string') {
      message = selectedError;
    }
    otherComp = <Alert type="error" message={message} />;
  } else if (!selectedData || selectedData.length === 0) {
    otherComp = <Alert type="error" message="Selection not found" />;
  }

  const loading = useMemo(
    () =>
      Array.from({ length: 5 }, (_, i) => (
        <tr key={i} className="divide-x divide-orange-300">
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden md:table-cell">
            <div className="h-5 w-28 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-64 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden lg:table-cell">
            <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden xl:table-cell">
            <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden md:table-cell">
            <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden xl:table-cell">
            <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap w-0 py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-20 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
        </tr>
      )),
    [],
  );

  let searchOtherComp = undefined;

  if (searchIsLoading) {
    searchOtherComp = loading;
  } else if (searchError) {
    let message = 'Something went wrong fetching data';
    if (typeof searchError === 'string') {
      message = searchError;
    }
    searchOtherComp = (
      <tr className="">
        <td
          colSpan={8}
          className="whitespace-nowrap py-4 pl-4 pr-4 font-bold text-red-600 text-center text-xl"
        >
          {message}
        </td>
      </tr>
    );
  } else if (!searchData || searchData.length === 0) {
    searchOtherComp = (
      <tr className="">
        <td
          colSpan={8}
          className="whitespace-nowrap py-4 pl-4 pr-4 font-bold text-gray-600 text-center text-xl"
        >
          No data available
        </td>
      </tr>
    );
  }

  let hasFilters = false;
  if (beaconStateSelectorSlot || beaconStateSelectorStateRoot || beaconStateSelectorId) {
    hasFilters = true;
  }

  return (
    <div className="bg-white/35 my-10 px-8 py-5 rounded-xl border-2 border-amber-200">
      <div className="absolute -mt-8 bg-white px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold border-2 border-sky-400">
        Beacon state
      </div>
      {hasFilters && (
        <button
          className="absolute right-8 sm:right-14 -mt-8 bg-white px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-gray-600 font-bold flex cursor-pointer transition hover:text-gray-800 border-2 border-gray-500 hover:border-gray-700"
          onClick={() => {
            setValue('beaconStateSelectorId', '');
            setValue('beaconStateSelectorSlot', '');
            setValue('beaconStateSelectorStateRoot', '');
          }}
        >
          Clear
          <XMarkIcon className="w-4 h-4" />
        </button>
      )}
      {beaconStateSelectorId &&
        (otherComp ?? (
          <dl className="grid grid-cols-1 sm:grid-cols-3 mt-5">
            <div className="px-4 sm:col-span-1 sm:px-0 pb-4 sm:pt-4">
              <dt className="text-sm font-bold leading-6 text-gray-700">Node</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">{state?.node}</dd>
            </div>
            <div className="border-t sm:border-none border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Slot</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">{state?.slot}</dd>
            </div>
            <div className="border-t sm:border-none border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Epoch</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">{state?.epoch}</dd>
            </div>
            <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">
                Beacon node Implementation
              </dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">
                {state?.beacon_implementation}
              </dd>
            </div>
            <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Node version</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">
                {state?.node_version}
              </dd>
            </div>
            <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">State root</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 font-mono truncate">
                {state?.state_root}
              </dd>
            </div>
          </dl>
        ))}
      {!beaconStateSelectorId && (
        <>
          <h3 className="text-lg font-bold my-5 text-gray-700">
            Search for a slot or state root to select a beacon state
          </h3>
          <div className="bg-white/35 border-lg rounded-lg p-4 border-2 border-amber-100">
            <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
              <div className="sm:col-span-3 sm:col-start-1">
                <label
                  htmlFor="beaconStateSelectorSlot"
                  className="block text-sm font-bold leading-6 text-gray-700"
                >
                  Slot
                </label>
                <div className="mt-2">
                  <Controller
                    control={control}
                    name="beaconStateSelectorSlot"
                    render={(props) => (
                      <DebouncedInput<'beaconStateSelectorSlot'>
                        controllerProps={props}
                        type="text"
                        name="beaconStateSelectorSlot"
                        className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
                      />
                    )}
                  />
                </div>
              </div>

              <div className="sm:col-span-3">
                <label
                  htmlFor="beaconStateSelectorStateRoot"
                  className="block text-sm font-bold leading-6 text-gray-700"
                >
                  State root
                </label>
                <div className="mt-2">
                  <Controller
                    control={control}
                    name="beaconStateSelectorStateRoot"
                    render={(props) => (
                      <DebouncedInput<'beaconStateSelectorStateRoot'>
                        controllerProps={props}
                        type="text"
                        name="beaconStateSelectorStateRoot"
                        className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
                      />
                    )}
                  />
                </div>
              </div>
            </div>
          </div>
        </>
      )}
      {!beaconStateSelectorId && (beaconStateSelectorSlot || beaconStateSelectorStateRoot) && (
        <div className="mt-8 flow-root">
          <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
            <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
              <table className="min-w-full divide-y divide-orange-500 sm:rounded-lg bg-white/55 shadow overflow-hidden border-2 border-amber-100">
                <thead className="bg-sky-400">
                  <tr className="divide-x divide-orange-300">
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50"
                    ></th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden md:table-cell"
                    >
                      <div className="flex">
                        <span className="whitespace-nowrap">Fetched at</span>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0"
                    >
                      <div className="flex">
                        <span className="whitespace-nowrap">Node</span>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden lg:table-cell"
                    >
                      <div className="flex">
                        <span className="whitespace-nowrap">Beacon node Implementation</span>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden xl:table-cell"
                    >
                      <div className="flex">
                        <span className="whitespace-nowrap">Node version</span>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden md:table-cell"
                    >
                      <div className="flex">
                        <span className="whitespace-nowrap">Epoch</span>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0"
                    >
                      <div className="flex">
                        <span className="whitespace-nowrap">Slot</span>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 hidden xl:table-cell"
                    >
                      <div className="flex">
                        <span className="whitespace-nowrap">State root</span>
                      </div>
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-orange-300">
                  {searchOtherComp
                    ? searchOtherComp
                    : searchData?.map((row) => (
                        <tr
                          key={row.id}
                          className="divide-x divide-orange-300 cursor-pointer"
                          onClick={() => {
                            setValue('beaconStateSelectorId', row.id);
                            setValue('beaconStateSelectorSlot', '');
                            setValue('beaconStateSelectorStateRoot', '');
                          }}
                        >
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-1">
                            <button
                              onClick={() => {
                                setValue('beaconStateSelectorId', row.id);
                                setValue('beaconStateSelectorSlot', '');
                                setValue('beaconStateSelectorStateRoot', '');
                              }}
                              className="rounded-md bg-white/35 px-2.5 py-1.5 text-sm font-semibold text-gray-700  hover:bg-gray-50 border-2 border-sky-400 hover:border-sky-600"
                            >
                              Select
                            </button>
                          </td>
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-0 hidden md:table-cell">
                            <span className="underline decoration-dotted underline-offset-2 cursor-help">
                              <TimeAgo date={new Date(row.fetched_at)} />
                            </span>
                          </td>
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                            <div className=" w-fit">
                              <span className="relative top-1 group transition">
                                {row.node}
                                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                              </span>
                            </div>
                          </td>
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden lg:table-cell">
                            <div className=" w-fit">
                              <span className="relative top-1 group transition">
                                {row.beacon_implementation}
                                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                              </span>
                            </div>
                          </td>
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden xl:table-cell">
                            <div className=" w-fit">
                              <span className="relative top-1 group transition">
                                {row.node_version}
                                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                              </span>
                            </div>
                          </td>
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden md:table-cell">
                            <div className=" w-fit">
                              <span className="relative top-1 group transition">
                                {row.epoch}
                                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                              </span>
                            </div>
                          </td>
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                            <div className=" w-fit">
                              <span className="relative top-1 group transition">
                                {row.slot}
                                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                              </span>
                            </div>
                          </td>
                          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden xl:table-cell">
                            <div className=" w-fit">
                              <span className="relative top-1 group transition font-mono">
                                {row.state_root}
                                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                              </span>
                            </div>
                          </td>
                        </tr>
                      ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
