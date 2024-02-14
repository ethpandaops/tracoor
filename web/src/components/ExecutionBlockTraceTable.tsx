import { useState, useMemo } from 'react';

import {
  ArrowDownTrayIcon,
  ArrowUpIcon,
  ArrowDownIcon,
  ArrowsUpDownIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import classNames from 'classnames';
import { useFormContext } from 'react-hook-form';
import TimeAgo from 'react-timeago';

import Pagination from '@components/Pagination';
import useNetwork from '@contexts/network';
import { useExecutionBlockTraces, useExecutionBlockTracesCount } from '@hooks/useQuery';

type SortConfig = {
  key: string;
  direction: 'ASC' | 'DESC';
};

export default function ExecutionBlockTraceTable() {
  const { network } = useNetwork();
  const [sortConfig, setSortConfig] = useState<SortConfig>({
    key: 'fetched_at',
    direction: 'DESC',
  });
  const { watch, setValue } = useFormContext();
  const [selectedTraces, setSelectedTraces] = useState<number[]>([]);
  const [selectedBlockHash, setSelectedBlockHash] = useState<string | undefined>();
  const [currentPage, setCurrentPage] = useState<number>(1);
  const itemsPerPage = 100;

  const handleSort = (key: string) => {
    setSortConfig((currentSortConfig: SortConfig) => {
      if (currentSortConfig.key === key) {
        return {
          key,
          direction: currentSortConfig.direction === 'ASC' ? 'DESC' : 'ASC',
        };
      }
      return { key, direction: 'DESC' };
    });
  };

  const [
    executionBlockTraceBlockHash,
    executionBlockTraceBlockNumber,
    executionBlockTraceNode,
    executionBlockTraceNodeImplementation,
    executionBlockTraceNodeVersion,
  ] = watch([
    'executionBlockTraceBlockHash',
    'executionBlockTraceBlockNumber',
    'executionBlockTraceNode',
    'executionBlockTraceNodeImplementation',
    'executionBlockTraceNodeVersion',
  ]);

  const { data, isLoading, error } = useExecutionBlockTraces({
    network: network ? network : undefined,
    block_hash: executionBlockTraceBlockHash ? executionBlockTraceBlockHash : undefined,
    block_number: executionBlockTraceBlockNumber
      ? parseInt(executionBlockTraceBlockNumber)
      : undefined,
    node: executionBlockTraceNode ? executionBlockTraceNode : undefined,
    node_version: executionBlockTraceNodeVersion ? executionBlockTraceNodeVersion : undefined,
    execution_implementation: executionBlockTraceNodeImplementation
      ? executionBlockTraceNodeImplementation
      : undefined,
    pagination: {
      limit: itemsPerPage,
      offset: (currentPage - 1) * itemsPerPage,
      order_by: `${sortConfig.key} ${sortConfig.direction}`,
    },
  });

  const { data: count } = useExecutionBlockTracesCount({
    network: network ? network : undefined,
    block_hash: executionBlockTraceBlockHash ? executionBlockTraceBlockHash : undefined,
    block_number: executionBlockTraceBlockNumber
      ? parseInt(executionBlockTraceBlockNumber)
      : undefined,
    node: executionBlockTraceNode ? executionBlockTraceNode : undefined,
    node_version: executionBlockTraceNodeVersion ? executionBlockTraceNodeVersion : undefined,
    execution_implementation: executionBlockTraceNodeImplementation
      ? executionBlockTraceNodeImplementation
      : undefined,
  });

  const totalPages = count ? Math.ceil(count / itemsPerPage) : 0;

  const loading = useMemo(
    () =>
      Array.from({ length: itemsPerPage }, (_, i) => (
        <tr key={i} className="divide-x divide-orange-300">
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600"></td>
        </tr>
      )),
    [],
  );

  let otherComp = undefined;

  // if loading or has error or has no data
  if (isLoading) {
    otherComp = loading;
  } else if (error) {
    let message = 'Something went wrong fetching data';
    if (typeof error === 'string') {
      message = error;
    }
    otherComp = (
      <tr className="">
        <td
          colSpan={7}
          className="whitespace-nowrap py-4 pl-4 pr-4 font-bold text-red-600 text-center text-xl"
        >
          {message}
        </td>
      </tr>
    );
  } else if (!data || data.length === 0) {
    otherComp = (
      <tr className="">
        <td
          colSpan={7}
          className="whitespace-nowrap py-4 pl-4 pr-4 font-bold text-gray-600 text-center text-xl"
        >
          No data available
        </td>
      </tr>
    );
  }

  const goevmlabsCmd = useMemo(() => {
    if (selectedTraces.length === 0) {
      return (
        <span className="text-gray-100">
          Select two traces to generate command to compare via{' '}
          <a
            className="underline text-sky-600 hover:text-sky-400"
            href="https://github.com/holiman/goevmlab"
          >
            goEVMLab
          </a>
        </span>
      );
    }

    if (selectedTraces.length === 1) {
      return (
        <div className="flex items-start">
          <span className="text-gray-100">
            Select one more to compare via{' '}
            <a
              className="underline text-sky-600 hover:text-sky-400"
              href="https://github.com/holiman/goevmlab"
            >
              goEVMLab
            </a>
          </span>
          <span
            className="ml-2 px-2 bg-white/15 rounded-lg cursor-pointer hover:bg-white/25 font-bold flex items-center text-gray-600 hover:text-gray-700 shadow-sm hover:shadow-md"
            onClick={() => {
              setSelectedTraces([]);
              setSelectedBlockHash(undefined);
            }}
          >
            <XMarkIcon className="pt-0.5 h-4 w-4" /> clear
          </span>
        </div>
      );
    }

    return (
      <div className="flex items-start">
        <div>
          <div className="bg-gray-900/20 p-2 rounded-xl">
            <div className="text-gray-100 text-left">
              <pre># download the traces</pre>
              <pre>
                wget -O {selectedTraces[0]}.json -q {window.location.origin}
                /download/execution_block_trace/{selectedTraces[0]}
              </pre>
              <pre>
                wget -O {selectedTraces[1]}.json -q {window.location.origin}
                /download/execution_block_trace/{selectedTraces[1]}
              </pre>
              <pre>
                goevmlab compare --state {selectedTraces[0]}.json --state {selectedTraces[1]}.json
              </pre>
            </div>
          </div>
        </div>
        <span
          className="ml-2 px-2 bg-white/15 rounded-lg cursor-pointer hover:bg-white/25 font-bold flex items-center text-gray-600 hover:text-gray-700 shadow-sm hover:shadow-md"
          onClick={() => {
            setSelectedTraces([]);
            setSelectedBlockHash(undefined);
          }}
        >
          <XMarkIcon className="pt-0.5 h-4 w-4" /> clear
        </span>
      </div>
    );
  }, [selectedTraces]);

  return (
    <>
      <div className="mt-8 flow-root">
        <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
          <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
            <div className="float-right bg-white/35 p-2 rounded-xl mb-5 text-right">
              {goevmlabsCmd}
            </div>
            <table className="min-w-full divide-y divide-orange-500 sm:rounded-lg bg-white/55 shadow overflow-hidden">
              <thead className="bg-sky-400">
                <tr className="divide-x divide-orange-300">
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'fetched_at' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'fetched_at' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'fetched_at' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50',
                    )}
                    onClick={() => handleSort('fetched_at')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Fetched at</span>
                      <span>
                        {sortConfig.key === 'fetched_at' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'node' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'node' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'node' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50',
                    )}
                    onClick={() => handleSort('node')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Node</span>
                      <span>
                        {sortConfig.key === 'node' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'execution_implementation' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'execution_implementation' &&
                        sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'execution_implementation' &&
                        sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50',
                    )}
                    onClick={() => handleSort('execution_implementation')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Beacon node Implementation</span>
                      <span>
                        {sortConfig.key === 'execution_implementation' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'node_version' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'node_version' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'node_version' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50',
                    )}
                    onClick={() => handleSort('node_version')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Node version</span>
                      <span>
                        {sortConfig.key === 'node_version' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'block_hash' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'block_hash' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'block_hash' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50',
                    )}
                    onClick={() => handleSort('block_hash')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Block hash</span>
                      <span>
                        {sortConfig.key === 'block_hash' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'block_number' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'block_number' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'block_number' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-10',
                    )}
                    onClick={() => handleSort('block_number')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Block number</span>
                      <span>
                        {sortConfig.key === 'block_number' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50"
                  ></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-orange-300">
                {otherComp
                  ? otherComp
                  : data?.map((row) => (
                      <tr
                        key={row.id}
                        className={classNames(
                          selectedTraces.includes(row.id) ? 'bg-white/25' : '',
                          'divide-x divide-orange-300',
                        )}
                      >
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-0">
                          <span className="underline decoration-dotted underline-offset-2 cursor-help">
                            <TimeAgo date={new Date(row.fetched_at)} />
                          </span>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                          <span
                            className="cursor-pointer hover:underline"
                            onClick={() => setValue('executionBlockTraceNode', row.node)}
                          >
                            {row.node}
                          </span>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                          <span
                            className="cursor-pointer hover:underline"
                            onClick={() =>
                              setValue(
                                'executionBlockTraceNodeImplementation',
                                row.execution_implementation,
                              )
                            }
                          >
                            {row.execution_implementation}
                          </span>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                          <span
                            className="cursor-pointer hover:underline"
                            onClick={() =>
                              setValue('executionBlockTraceNodeVersion', row.node_version)
                            }
                          >
                            {row.node_version}
                          </span>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                          <span
                            className="cursor-pointer hover:underline"
                            onClick={() => setValue('executionBlockTraceBlockHash', row.block_hash)}
                          >
                            {row.block_hash}
                          </span>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                          <span
                            className="cursor-pointer hover:underline"
                            onClick={() =>
                              setValue('executionBlockTraceBlockNumber', row.block_number)
                            }
                          >
                            {row.block_number}
                          </span>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-1">
                          <div className="flex flex-row ">
                            <a
                              href={`/download/execution_block_trace/${row.id}`}
                              download={`execution_block_trace_${row.id}.json`}
                              className="text-sky-500 hover:text-sky-600 px-2"
                            >
                              <ArrowDownTrayIcon className="h-6 w-6" aria-hidden="true" />
                            </a>
                            {(selectedTraces.length > 1 && selectedTraces.includes(row.id)) ||
                            (selectedTraces.length <= 1 &&
                              (!selectedBlockHash || row.block_hash === selectedBlockHash)) ? (
                              <button
                                type="button"
                                className={classNames(
                                  selectedTraces.includes(row.id)
                                    ? 'bg-white/45 hover:bg-white/65'
                                    : '',
                                  'rounded bg-white/25 px-2 py-1 text-xs font-semibold text-sky-500 shadow-sm hover:bg-white/55',
                                )}
                                onClick={() => {
                                  if (selectedTraces.includes(row.id)) {
                                    if (selectedTraces.length === 1) {
                                      setSelectedBlockHash(undefined);
                                    }
                                    setSelectedTraces(selectedTraces.filter((id) => id !== row.id));
                                  } else {
                                    setSelectedBlockHash(row.block_hash);
                                    setSelectedTraces([...selectedTraces, row.id]);
                                  }
                                }}
                              >
                                {selectedTraces.includes(row.id) ? 'Deselect' : 'Compare'}
                              </button>
                            ) : null}
                          </div>
                        </td>
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <div className={classNames(totalPages <= 1 ? 'hidden' : '', 'mt-10 mb-20')}>
        <Pagination
          currentPage={currentPage}
          totalPages={totalPages}
          onPageChange={(page: number) => setCurrentPage(page)}
        />
      </div>
    </>
  );
}
