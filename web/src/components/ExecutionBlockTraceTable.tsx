import { useState, useMemo } from 'react';

import { ArrowDownTrayIcon } from '@heroicons/react/24/outline';
import classNames from 'classnames';
import { useFormContext } from 'react-hook-form';

import { useExecutionBlockTraces } from '@hooks/useQuery';

export default function ExecutionBlockTraceTable() {
  const { watch, setValue } = useFormContext();
  const [selectedTraces, setSelectedTraces] = useState<number[]>([]);

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

  // fetchListBeaconState
  const { data, isLoading, error } = useExecutionBlockTraces({
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
      limit: 10,
      offset: 0,
      order_by: 'fetched_at DESC',
    },
  });

  const loading = useMemo(
    () =>
      Array.from({ length: 10 }, (_, i) => (
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
        <span className="text-gray-100">
          Select one more to compare via{' '}
          <a
            className="underline text-sky-600 hover:text-sky-400"
            href="https://github.com/holiman/goevmlab"
          >
            goEVMLab
          </a>
        </span>
      );
    }

    return (
      <div className="bg-gray-900/20 p-2 rounded-xl">
        <div className="text-gray-100">
          goevmlab compare --state {selectedTraces[0]} --state {selectedTraces[1]}
        </div>
      </div>
    );
  }, [selectedTraces]);

  return (
    <>
      <div className="float-right bg-white/35 p-2 rounded-xl mb-5 text-right">{goevmlabsCmd}</div>
      <table className="min-w-full divide-y divide-orange-500 sm:rounded-lg bg-white/55 shadow overflow-hidden">
        <thead className="bg-sky-400">
          <tr className="divide-x divide-orange-300">
            <th
              scope="col"
              className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50"
            >
              Fetched at
            </th>
            <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
              Node
            </th>
            <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
              Beacon node Implementation
            </th>
            <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
              Node version
            </th>
            <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
              Block hash
            </th>
            <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
              Block number
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
                  <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                    <span className="">{new Date(row.fetched_at).toISOString()}</span>
                  </td>
                  <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                    <span
                      className="cursor-pointer"
                      onClick={() => setValue('executionBlockTraceNode', row.node)}
                    >
                      {row.node}
                    </span>
                  </td>
                  <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                    <span
                      className="cursor-pointer"
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
                      className="cursor-pointer"
                      onClick={() => setValue('executionBlockTraceNodeVersion', row.node_version)}
                    >
                      {row.node_version}
                    </span>
                  </td>
                  <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                    <span
                      className="cursor-pointer"
                      onClick={() => setValue('executionBlockTraceEpoch', row.block_hash)}
                    >
                      {row.block_hash}
                    </span>
                  </td>
                  <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                    <span
                      className="cursor-pointer"
                      onClick={() => setValue('executionBlockTraceSlot', row.block_number)}
                    >
                      {row.block_number}
                    </span>
                  </td>
                  <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 flex flex-row max-w-10">
                    <a href="#" className="text-sky-500 hover:text-sky-600 px-2">
                      <ArrowDownTrayIcon className="h-6 w-6" aria-hidden="true" />
                    </a>
                    {(selectedTraces.length > 1 && selectedTraces.includes(row.id)) ||
                    selectedTraces.length <= 1 ? (
                      <button
                        type="button"
                        className={classNames(
                          selectedTraces.includes(row.id) ? 'bg-white/45 hover:bg-white/65' : '',
                          'rounded bg-white/25 px-2 py-1 text-xs font-semibold text-sky-500 shadow-sm hover:bg-white/55',
                        )}
                        onClick={() => {
                          if (selectedTraces.includes(row.id)) {
                            setSelectedTraces(selectedTraces.filter((id) => id !== row.id));
                          } else {
                            setSelectedTraces([...selectedTraces, row.id]);
                          }
                        }}
                      >
                        {selectedTraces.includes(row.id) ? 'Deselect' : 'Compare'}
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
        </tbody>
      </table>
    </>
  );
}
