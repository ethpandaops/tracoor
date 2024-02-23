import { ArrowDownTrayIcon, ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import TimeAgo from 'react-timeago';
import { useLocation } from 'wouter';

import CopyToClipboard from '@components/CopyToClipboard';
import ErrorAlert from '@components/ErrorAlert';
import ExecutionBlockTraceEVMLabs from '@components/ExecutionBlockTraceEVMLabs';
import Loading from '@components/Loading';
import useNetwork from '@contexts/network';
import { useExecutionBlockTraces } from '@hooks/useQuery';

export default function ExecutionBlockTraceId({ id }: { id: string }) {
  const [, setLocation] = useLocation();
  const { setValue } = useFormContext();
  const { network } = useNetwork();
  const { data, isLoading, error } = useExecutionBlockTraces({
    network: network ? network : undefined,
    id: id,
    pagination: {
      limit: 1,
    },
  });

  const trace = data?.[0];

  const {
    data: relatedData,
    isLoading: relatedIsLoading,
    error: relatedError,
  } = useExecutionBlockTraces(
    {
      network: network ? network : undefined,
      block_hash: trace?.block_hash,
    },
    Boolean(trace?.block_hash),
  );

  const relatedTraces = relatedData?.filter((t) => t.id !== id);

  const handleSearch = (key: string, value: unknown) => {
    setValue(key, value);
    setLocation('/execution_block_trace');
  };

  let errorMessage = undefined;

  if (error) {
    errorMessage = 'Error fetching bad block';
    if (error instanceof Error) {
      errorMessage = `Error fetching bad block: ${error.message}`;
    }
  } else if (!isLoading && !trace) {
    errorMessage = 'Bad block not found';
  }

  let evmLabsComp = undefined;
  if (relatedError) {
    let message = 'Error fetching related traces';
    if (relatedError instanceof Error) {
      message = `Error fetching related traces: ${relatedError.message}`;
    }
    evmLabsComp = <ErrorAlert message={message} />;
  } else if (trace?.block_hash && !relatedIsLoading && !relatedTraces?.length) {
    evmLabsComp = (
      <div className="p-5 text-gray-600">
        Need more than one block trace for block hash{' '}
        <span className="text-sky-500">{trace?.block_hash}</span>
      </div>
    );
  } else if (!relatedIsLoading && relatedTraces?.length && trace) {
    evmLabsComp = <ExecutionBlockTraceEVMLabs primaryTrace={trace} relatedTraces={relatedTraces} />;
  } else {
    evmLabsComp = <Loading className="m-5" />;
  }

  if (errorMessage) {
    return (
      <div className="bg-gray-50 dark:bg-gray-800 shadow dark:shadow-inner">
        <div className="border-t border-gray-200 dark:border-b dark:border-gray-800 px-4 py-5 sm:p-0">
          <dl className="sm:divide-y sm:divide-gray-200 dark:divide-gray-900">
            <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">ID</dt>
              <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">{id}</dd>
            </div>
            <ErrorAlert message={errorMessage} />
          </dl>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-50 dark:bg-gray-800 shadow dark:shadow-inner">
      <div className="border-t border-gray-200 dark:border-b dark:border-gray-800 px-4 py-5 sm:p-0">
        <dl className="sm:divide-y sm:divide-gray-200 dark:divide-gray-900">
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">ID</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">{id}</dd>
          </div>
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Fetched at</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 underline decoration-dotted underline-offset-2 cursor-help">
              {isLoading ? (
                <div className="h-5 w-32 bg-gray-600/35 rounded-xl animate-pulse"></div>
              ) : (
                trace && <TimeAgo date={new Date(trace.fetched_at)} />
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Network</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              {isLoading ? (
                <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
              ) : (
                trace?.network
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Node</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('executionBlockTraceNode', trace?.node)}
              >
                {isLoading ? (
                  <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  trace?.node
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Execution Implementation</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() =>
                  handleSearch(
                    'executionBlockTraceNodeImplementation',
                    trace?.execution_implementation,
                  )
                }
              >
                {isLoading ? (
                  <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  trace?.execution_implementation
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Node version</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('executionBlockTraceNodeVersion', trace?.node_version)}
              >
                {isLoading ? (
                  <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  trace?.node_version
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Block number</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('executionBlockTraceBlockNumber', trace?.block_number)}
              >
                {isLoading ? (
                  <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  trace?.block_number
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
              <CopyToClipboard
                text={trace?.block_number.toString() ?? ''}
                className="ml-2 hidden lg:block"
              />
            </dd>
          </div>
          <div className="py-4 sm:py-5 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Block Hash</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              <span className="lg:hidden font-mono flex">
                <span
                  className="relative top-1 group transition duration-300 cursor-pointer"
                  onClick={() => handleSearch('executionBlockTraceBlockHash', trace?.block_hash)}
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    trace?.block_hash
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-400"></span>
                </span>
              </span>
              <span className="hidden lg:flex font-mono">
                <span
                  className="relative top-1 group transition cursor-pointer"
                  onClick={() => handleSearch('executionBlockTraceBlockHash', trace?.block_hash)}
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    trace?.block_hash
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
                </span>
                <CopyToClipboard text={trace?.block_hash ?? ''} className="ml-2" />
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 sm:px-6 flex justify-center sm:bg-gray-100 shadow-sm">
            <dt className="text-md text-gray-500 font-bold">
              <a
                href={`/download/execution_block_trace/${id}`}
                download={
                  trace
                    ? `execution_block_trace-${trace.block_number}-${trace.block_hash}-${trace.node}.json.gz`
                    : `execution_block_trace-${id}.json.gz`
                }
                className="text-amber-500 hover:text-amber-600 px-2 flex"
              >
                Download <ArrowDownTrayIcon className="w-6 h-6 ml-2" />
              </a>
            </dt>
          </div>
          <div className="py-4 sm:py-5 sm:px-6 flex justify-center sm:justify-start bg-amber-500/85">
            <dt className="text-md text-white font-bold flex items-center">
              EVM laboratory transaction tracediff{' '}
              <a
                target="_blank"
                href="https://github.com/holiman/goevmlab/tree/master/cmd/tracediff"
                rel="noreferrer"
              >
                <ArrowTopRightOnSquareIcon className="w-4 h-4 ml-2" />
              </a>
            </dt>
          </div>
          <div>{evmLabsComp}</div>
        </dl>
      </div>
    </div>
  );
}
