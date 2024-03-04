import { useMemo } from 'react';

import { XMarkIcon } from '@heroicons/react/24/outline';
import { useFormContext, Controller } from 'react-hook-form';
import TimeAgo from 'react-timeago';

import Alert from '@components/Alert';
import DebouncedInput from '@components/DebouncedInput';
import Loading from '@components/Loading';
import useNetwork from '@contexts/network';
import { useExecutionBlockTraces } from '@hooks/useQuery';

export default function ExecutionBlockTraceSelector({
  num,
  excludeNum,
}: {
  num?: number;
  excludeNum?: number;
}) {
  const { control, watch, setValue } = useFormContext();
  const { network } = useNetwork();

  const [
    executionBlockTraceSelectorId,
    executionBlockTraceSelectorBlockHash,
    executionBlockTraceSelectorBlockNumber,
    excludeId,
  ] = watch([
    `executionBlockTraceSelectorId${num}`,
    `executionBlockTraceSelectorBlockHash${num}`,
    `executionBlockTraceSelectorBlockNumber${num}`,
    `executionBlockTraceSelectorId${excludeNum}`,
  ]);

  const {
    data: selectedData,
    isLoading: selectedIsLoading,
    error: selectedError,
  } = useExecutionBlockTraces(
    {
      network: network ? network : undefined,
      id: executionBlockTraceSelectorId,
      pagination: {
        limit: 1,
      },
    },
    Boolean(executionBlockTraceSelectorId),
  );

  const trace = selectedData?.[0];

  const {
    data: searchData,
    isLoading: searchIsLoading,
    error: searchError,
  } = useExecutionBlockTraces(
    {
      network: network ? network : undefined,
      block_hash: executionBlockTraceSelectorBlockHash
        ? executionBlockTraceSelectorBlockHash
        : undefined,
      block_number: executionBlockTraceSelectorBlockNumber
        ? executionBlockTraceSelectorBlockNumber
        : undefined,
    },
    Boolean(
      !executionBlockTraceSelectorId &&
        (executionBlockTraceSelectorBlockHash || executionBlockTraceSelectorBlockNumber),
    ),
  );

  const filteredSearchData = useMemo(() => {
    if (excludeId && searchData) {
      return searchData.filter((row) => row.id !== excludeId);
    }
    return searchData;
  }, [excludeId, searchData]);

  let otherComp = undefined;

  if (selectedIsLoading) {
    otherComp = (
      <dl className="grid grid-cols-1 sm:grid-cols-4 xl:grid-cols-5 mt-5">
        <div className="px-4 sm:col-span-1 sm:px-0 pb-4 sm:pt-4">
          <dt className="text-sm font-bold leading-6 text-gray-700">Node</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-64 sm:w-32 2xl:w-64 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Block Number</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-20 sm:w-32 2xl:w-24 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Execution Implementation</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-32 2xl:w-28 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="block sm:hidden xl:block px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Node version</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 h-5 w-64 sm:w-32 2xl:w-64 bg-gray-600/35 rounded-xl animate-pulse"></dd>
        </div>
        <div className="px-4 py-4 sm:col-span-1 sm:px-0">
          <dt className="text-sm font-bold leading-6 text-gray-700">Block hash</dt>
          <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 font-mono truncate h-5 w-64 sm:w-32 2xl:w-64 bg-gray-600/35 rounded-xl animate-pulse"></dd>
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
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 2xl:table-cell">
            <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 4xl:table-cell">
            <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 3xl:table-cell">
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
  } else if (!filteredSearchData || filteredSearchData.length === 0) {
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
  if (
    executionBlockTraceSelectorBlockHash ||
    executionBlockTraceSelectorBlockNumber ||
    executionBlockTraceSelectorId
  ) {
    hasFilters = true;
  }

  return (
    <div className="bg-white/35 my-10 px-8 py-5 rounded-xl">
      <div className="absolute -mt-8 bg-white/65 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold border-2 border-sky-400">
        Execution Block Trace{num ? ` #${num}` : ''}
      </div>
      {hasFilters && (
        <button
          className="absolute right-8 sm:right-14 -mt-8 bg-white/85 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-gray-600 font-bold flex cursor-pointer transition hover:text-gray-800 border-2 border-gray-500 hover:border-gray-700"
          onClick={() => {
            setValue(`executionBlockTraceSelectorId${num}`, '');
            setValue(`executionBlockTraceSelectorBlockHash${num}`, '');
            setValue(`executionBlockTraceSelectorBlockNumber${num}`, '');
          }}
        >
          Clear
          <XMarkIcon className="w-4 h-4" />
        </button>
      )}
      {executionBlockTraceSelectorId &&
        (otherComp ?? (
          <dl className="grid grid-cols-1 sm:grid-cols-4 xl:grid-cols-5 mt-5">
            <div className="px-4 sm:col-span-1 sm:px-0 pb-4 sm:pt-4">
              <dt className="text-sm font-bold leading-6 text-gray-700">Node</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">{trace?.node}</dd>
            </div>
            <div className="px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Block Number</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">
                {trace?.block_number}
              </dd>
            </div>
            <div className="px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">
                Execution Implementation
              </dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">
                {trace?.execution_implementation}
              </dd>
            </div>
            <div className="block sm:hidden xl:block px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Node version</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1">
                {trace?.node_version}
              </dd>
            </div>
            <div className="px-4 py-4 sm:col-span-1 sm:px-0">
              <dt className="text-sm font-bold leading-6 text-gray-700">Block hash</dt>
              <dd className="mt-0.5 text-sm leading-6 text-gray-700 sm:mt-1 font-mono truncate">
                {trace?.block_hash}
              </dd>
            </div>
          </dl>
        ))}
      {!executionBlockTraceSelectorId && (
        <>
          <h3 className="text-lg font-bold my-5 text-gray-700">
            Search for a block hash or number to select a execution block trace
          </h3>
          <div className="bg-white/35 border-lg rounded-lg p-4">
            <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
              <div className="sm:col-span-3 sm:col-start-1">
                <label
                  htmlFor={`executionBlockTraceSelectorBlockHash${num}`}
                  className="block text-sm font-bold leading-6 text-gray-700"
                >
                  Block hash
                </label>
                <div className="mt-2">
                  <Controller
                    control={control}
                    name={`executionBlockTraceSelectorBlockHash${num}`}
                    render={(props) => (
                      <DebouncedInput
                        controllerProps={props}
                        type="text"
                        name={`executionBlockTraceSelectorBlockHash${num}`}
                        className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
                      />
                    )}
                  />
                </div>
              </div>

              <div className="sm:col-span-3">
                <label
                  htmlFor={`executionBlockTraceSelectorBlockNumber${num}`}
                  className="block text-sm font-bold leading-6 text-gray-700"
                >
                  Block number
                </label>
                <div className="mt-2">
                  <Controller
                    control={control}
                    name={`executionBlockTraceSelectorBlockNumber${num}`}
                    render={(props) => (
                      <DebouncedInput
                        controllerProps={props}
                        type="text"
                        name={`executionBlockTraceSelectorBlockNumber${num}`}
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
      {!executionBlockTraceSelectorId &&
        (executionBlockTraceSelectorBlockHash || executionBlockTraceSelectorBlockNumber) && (
          <div className="mt-8 flow-root">
            <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
              <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
                <table className="min-w-full divide-y divide-orange-500 sm:rounded-lg bg-white/55 shadow overflow-hidden">
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
                        className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden 2xl:table-cell"
                      >
                        <div className="flex">
                          <span className="whitespace-nowrap">Beacon node Implementation</span>
                        </div>
                      </th>
                      <th
                        scope="col"
                        className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden 4xl:table-cell"
                      >
                        <div className="flex">
                          <span className="whitespace-nowrap">Node version</span>
                        </div>
                      </th>
                      <th
                        scope="col"
                        className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0"
                      >
                        <div className="flex">
                          <span className="whitespace-nowrap">Block number</span>
                        </div>
                      </th>
                      <th
                        scope="col"
                        className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 hidden 3xl:table-cell"
                      >
                        <div className="flex">
                          <span className="whitespace-nowrap">Block hash</span>
                        </div>
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-orange-300">
                    {searchOtherComp
                      ? searchOtherComp
                      : filteredSearchData?.map((row) => (
                          <tr
                            key={row.id}
                            className="divide-x divide-orange-300 cursor-pointer"
                            onClick={() => {
                              setValue(`executionBlockTraceSelectorId${num}`, row.id);
                              setValue(`executionBlockTraceSelectorBlockHash${num}`, '');
                              setValue(`executionBlockTraceSelectorBlockNumber${num}`, '');
                            }}
                          >
                            <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-1">
                              <button
                                onClick={() => {
                                  setValue(`executionBlockTraceSelectorId${num}`, row.id);
                                  setValue(`executionBlockTraceSelectorBlockHash${num}`, '');
                                  setValue(`executionBlockTraceSelectorBlockNumber${num}`, '');
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
                            <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 2xl:table-cell">
                              <div className=" w-fit">
                                <span className="relative top-1 group transition">
                                  {row.execution_implementation}
                                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                                </span>
                              </div>
                            </td>
                            <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 4xl:table-cell">
                              <div className=" w-fit">
                                <span className="relative top-1 group transition">
                                  {row.node_version}
                                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                                </span>
                              </div>
                            </td>
                            <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                              <div className=" w-fit">
                                <span className="relative top-1 group transition">
                                  {row.block_number}
                                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                                </span>
                              </div>
                            </td>
                            <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 3xl:table-cell">
                              <div className=" w-fit">
                                <span className="relative top-1 group transition font-mono">
                                  {row.block_hash}
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
