import { useMemo } from 'react';

import { XMarkIcon } from '@heroicons/react/24/outline';
import { useFormContext, Controller } from 'react-hook-form';
import TimeAgo from 'react-timeago';
import { Link } from 'wouter';

import { BeaconBlock, BeaconBadBlock } from '@app/types/api';
import Alert from '@components/Alert';
import DebouncedInput from '@components/DebouncedInput';
import Loading from '@components/Loading';
import useNetwork from '@contexts/network';
import { useBeaconBlocks, useBeaconBadBlocks } from '@hooks/useQuery';

export default function BeaconBlockSelector() {
  const { control, watch, setValue } = useFormContext();
  const { network } = useNetwork();

  const [beaconBlockSelectorId, beaconBlockSelectorSlot, beaconBlockSelectorBlockRoot] = watch([
    'beaconBlockSelectorId',
    'beaconBlockSelectorSlot',
    'beaconBlockSelectorBlockRoot',
  ]);

  const {
    data: selectedData,
    isLoading: selectedIsLoading,
    error: selectedError,
  } = useBeaconBlocks(
    {
      network: network ? network : undefined,
      id: beaconBlockSelectorId,
      pagination: {
        limit: 1,
      },
    },
    Boolean(beaconBlockSelectorId),
  );

  const {
    data: selectedBadData,
    isLoading: selectedBadIsLoading,
    error: selectedBadError,
  } = useBeaconBadBlocks(
    {
      network: network ? network : undefined,
      id: beaconBlockSelectorId,
      pagination: {
        limit: 1,
      },
    },
    Boolean(beaconBlockSelectorId),
  );

  const block = selectedBadData?.[0] ?? selectedData?.[0];

  const {
    data: searchData,
    isLoading: searchIsLoading,
    error: searchError,
  } = useBeaconBlocks(
    {
      network: network ? network : undefined,
      slot: beaconBlockSelectorSlot ? beaconBlockSelectorSlot : undefined,
      block_root: beaconBlockSelectorBlockRoot ? beaconBlockSelectorBlockRoot : undefined,
    },
    Boolean(!beaconBlockSelectorId && (beaconBlockSelectorSlot || beaconBlockSelectorBlockRoot)),
  );

  const {
    data: searchBadData,
    isLoading: searchBadIsLoading,
    error: searchBadError,
  } = useBeaconBadBlocks(
    {
      network: network ? network : undefined,
      slot: beaconBlockSelectorSlot ? beaconBlockSelectorSlot : undefined,
      block_root: beaconBlockSelectorBlockRoot ? beaconBlockSelectorBlockRoot : undefined,
    },
    Boolean(!beaconBlockSelectorId && (beaconBlockSelectorSlot || beaconBlockSelectorBlockRoot)),
  );

  let otherComp = undefined;

  if (selectedIsLoading || selectedBadIsLoading) {
    otherComp = <Loading />;
  } else if (selectedError || selectedBadError) {
    let message = 'Something went wrong fetching data';
    if (typeof selectedError === 'string') {
      message = selectedError;
    }
    if (typeof selectedBadError === 'string') {
      message = selectedBadError;
    }
    otherComp = <Alert type="error" message={message} />;
  } else if (!block) {
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

  if (searchIsLoading || searchBadIsLoading) {
    searchOtherComp = loading;
  } else if (searchError || searchBadError) {
    let message = 'Something went wrong fetching data';
    if (typeof searchError === 'string') {
      message = searchError;
    }
    if (typeof searchBadError === 'string') {
      message = searchBadError;
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
  } else if (
    (!searchData || searchData.length === 0) &&
    (!searchBadData || searchBadData.length === 0)
  ) {
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
  if (beaconBlockSelectorSlot || beaconBlockSelectorBlockRoot || beaconBlockSelectorId) {
    hasFilters = true;
  }

  function generateRow(row: BeaconBlock | BeaconBadBlock, bad = false) {
    return (
      <tr key={row.id} className="divide-x divide-orange-300">
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
          <div className=" w-fit flex items-center">
            <span className="relative top-1 group transition font-mono">
              {row.block_root}
              <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
            </span>
            {bad && (
              <Link
                href={`/beacon_bad_block/${row.id}`}
                className="bg-red-600/50 text-xs text-gray-100 py-0.5 px-2 rounded-xl ml-2"
              >
                bad block
              </Link>
            )}
          </div>
        </td>
        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-1">
          <button
            onClick={() => {
              setValue('beaconBlockSelectorId', row.id);
              setValue('beaconBlockSelectorSlot', '');
              setValue('beaconBlockSelectorBlockRoot', '');
            }}
            className="rounded-md bg-white/35 px-2.5 py-1.5 text-sm font-semibold text-gray-700  hover:bg-gray-50"
          >
            Select
          </button>
        </td>
      </tr>
    );
  }

  return (
    <div className="bg-white/35 my-10 px-8 py-5 rounded-xl">
      <div className="absolute -mt-8 bg-white/65 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold">
        Beacon block
      </div>
      {hasFilters && (
        <button
          className="absolute right-14 -mt-8 bg-white/85 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-gray-600 font-bold flex cursor-pointer transition hover:text-gray-800"
          onClick={() => {
            setValue('beaconBlockSelectorId', '');
            setValue('beaconBlockSelectorSlot', '');
            setValue('beaconBlockSelectorBlockRoot', '');
          }}
        >
          Clear
          <XMarkIcon className="w-4 h-4" />
        </button>
      )}
      {beaconBlockSelectorId &&
        (otherComp ?? (
          <dl className="grid grid-cols-1 sm:grid-cols-3 mt-5">
            <div className="px-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Node</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">{block?.node}</dd>
            </div>
            <div className="border-t sm:border-none border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Slot</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">{block?.slot}</dd>
            </div>
            <div className="border-t sm:border-none border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Epoch</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">{block?.epoch}</dd>
            </div>
            <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">
                Beacon node Implementation
              </dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">
                {block?.beacon_implementation}
              </dd>
            </div>
            <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Node version</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">
                {block?.node_version}
              </dd>
            </div>
            <div className="border-t border-amber-500 px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Block root</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 font-mono truncate">
                {block?.block_root}
              </dd>
            </div>
          </dl>
        ))}
      {!beaconBlockSelectorId && (
        <>
          <h3 className="text-lg font-bold my-5 text-gray-700">
            Search for a slot or block root to select a beacon block
          </h3>
          <div className="bg-white/35 border-lg rounded-lg p-4">
            <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
              <div className="sm:col-span-3 sm:col-start-1">
                <label
                  htmlFor="beaconBlockSelectorSlot"
                  className="block text-sm font-bold leading-6 text-gray-700"
                >
                  Slot
                </label>
                <div className="mt-2">
                  <Controller
                    control={control}
                    name="beaconBlockSelectorSlot"
                    render={(props) => (
                      <DebouncedInput<'beaconBlockSelectorSlot'>
                        controllerProps={props}
                        type="text"
                        name="beaconBlockSelectorSlot"
                        className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
                      />
                    )}
                  />
                </div>
              </div>

              <div className="sm:col-span-3">
                <label
                  htmlFor="beaconBlockSelectorBlockRoot"
                  className="block text-sm font-bold leading-6 text-gray-700"
                >
                  Block root
                </label>
                <div className="mt-2">
                  <Controller
                    control={control}
                    name="beaconBlockSelectorBlockRoot"
                    render={(props) => (
                      <DebouncedInput<'beaconBlockSelectorBlockRoot'>
                        controllerProps={props}
                        type="text"
                        name="beaconBlockSelectorBlockRoot"
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
      {!beaconBlockSelectorId && (beaconBlockSelectorSlot || beaconBlockSelectorBlockRoot) && (
        <div className="mt-8 flow-root">
          <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
            <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
              <table className="min-w-full divide-y divide-orange-500 sm:rounded-lg bg-white/55 shadow overflow-hidden">
                <thead className="bg-sky-400">
                  <tr className="divide-x divide-orange-300">
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
                        <span className="whitespace-nowrap">Block root</span>
                      </div>
                    </th>
                    <th
                      scope="col"
                      className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50"
                    ></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-orange-300">
                  {searchOtherComp
                    ? searchOtherComp
                    : [
                        ...(searchBadData?.map((row) => generateRow(row, true)) ?? []),
                        ...(searchData?.map((row) => generateRow(row)) ?? []),
                      ]}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
