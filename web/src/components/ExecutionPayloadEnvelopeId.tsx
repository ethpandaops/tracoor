import { ArrowDownTrayIcon, ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import TimeAgo from 'react-timeago';
import { useLocation } from 'wouter';

import Alert from '@components/Alert';
import CopyToClipboard from '@components/CopyToClipboard';
import VerificationBadge from '@components/VerificationBadge';
import useNetwork from '@contexts/network';
import { useExecutionPayloadEnvelopes } from '@hooks/useQuery';

export default function ExecutionPayloadEnvelopeId({ id }: { id: string }) {
  const [, setLocation] = useLocation();
  const { setValue } = useFormContext();
  const { network } = useNetwork();
  const { data, isLoading, error } = useExecutionPayloadEnvelopes({
    network: network ? network : undefined,
    id: id,
    pagination: {
      limit: 1,
    },
  });

  const envelope = data?.[0];

  const handleSearch = (key: string, value: unknown) => {
    setValue(key, value);
    setLocation('/execution_payload_envelope');
  };

  // The envelope carries the payload its beacon block no longer does, so there has to be a
  // way across: without it the two halves of a gloas slot are unreachable from each other.
  const handleBlockSearch = (blockRoot: string) => {
    setValue('beaconBlockBlockRoot', blockRoot);
    setLocation('/beacon_block');
  };

  let errorMessage = undefined;

  if (error) {
    errorMessage = 'Error fetching execution payload envelope';
    if (error instanceof Error) {
      errorMessage = `Error fetching execution payload envelope: ${error.message}`;
    }
  } else if (!isLoading && !envelope) {
    errorMessage = 'Execution payload envelope not found';
  }

  if (errorMessage) {
    return (
      <div className="bg-gray-50 dark:bg-gray-800 shadow dark:shadow-inner">
        <div className="border-t border-gray-200 dark:border-b dark:border-gray-800">
          <dl className="sm:divide-y sm:divide-gray-200 dark:divide-gray-900">
            <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
              <dt className="text-sm font-medium text-gray-500">ID</dt>
              <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">{id}</dd>
            </div>
            <Alert type="error" message={errorMessage} />
          </dl>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-50 dark:bg-gray-800 shadow dark:shadow-inner">
      <div className="border-t border-gray-200 dark:border-b dark:border-gray-800">
        <dl className="sm:divide-y sm:divide-gray-200 dark:divide-gray-900">
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">ID</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">{id}</dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Fetched at</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 underline decoration-dotted underline-offset-2 cursor-help">
              {isLoading ? (
                <div className="h-5 w-32 bg-gray-600/35 rounded-xl animate-pulse"></div>
              ) : (
                envelope && <TimeAgo date={new Date(envelope.fetched_at)} />
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Network</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              {isLoading ? (
                <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
              ) : (
                envelope?.network
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Node</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('executionPayloadEnvelopeNode', envelope?.node)}
              >
                {isLoading ? (
                  <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  envelope?.node
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Execution Implementation</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() =>
                  handleSearch(
                    'executionPayloadEnvelopeNodeImplementation',
                    envelope?.beacon_implementation,
                  )
                }
              >
                {isLoading ? (
                  <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  envelope?.beacon_implementation
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Node version</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() =>
                  handleSearch('executionPayloadEnvelopeNodeVersion', envelope?.node_version)
                }
              >
                {isLoading ? (
                  <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  envelope?.node_version
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Epoch</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() =>
                  handleSearch('executionPayloadEnvelopeEpoch', envelope?.epoch.toString())
                }
              >
                {isLoading ? (
                  <div className="h-5 w-20 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  envelope?.epoch
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
              <CopyToClipboard
                text={envelope?.epoch.toString() ?? ''}
                className="ml-2 hidden lg:block"
              />
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Slot</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() =>
                  handleSearch('executionPayloadEnvelopeSlot', envelope?.slot.toString())
                }
              >
                {isLoading ? (
                  <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  envelope?.slot
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
              <CopyToClipboard
                text={envelope?.slot.toString() ?? ''}
                className="ml-2 hidden lg:block"
              />
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Block root</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              <span className="lg:hidden font-mono flex">
                <span
                  className="relative top-1 group transition duration-300 cursor-pointer"
                  onClick={() =>
                    handleSearch('executionPayloadEnvelopeBlockRoot', envelope?.block_root)
                  }
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    envelope?.block_root
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-400"></span>
                </span>
              </span>
              <span className="hidden lg:flex font-mono">
                <span
                  className="relative top-1 group transition cursor-pointer"
                  onClick={() =>
                    handleSearch('executionPayloadEnvelopeBlockRoot', envelope?.block_root)
                  }
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    envelope?.block_root
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
                </span>
                <CopyToClipboard text={envelope?.block_root ?? ''} className="ml-2" />
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Content hash</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              {isLoading ? (
                <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
              ) : (
                <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
                  <VerificationBadge
                    contentHash={envelope?.content_hash}
                    verifiedAt={envelope?.verified_at}
                    contentMatchedAt={envelope?.content_matched_at}
                    agreementCount={envelope?.agreement_count}
                  />
                  {envelope?.content_hash && (
                    <span className="flex font-mono">
                      <span className="relative top-1">{envelope.content_hash}</span>
                      <CopyToClipboard text={envelope.content_hash} className="ml-2" />
                    </span>
                  )}
                </div>
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:px-6 flex flex-wrap justify-center gap-x-6 gap-y-2 sm:bg-gray-100 sm:dark:bg-gray-900">
            {envelope?.block_root && (
              <dt className="text-md text-gray-500 font-bold">
                <button
                  type="button"
                  onClick={() => handleBlockSearch(envelope.block_root)}
                  className="text-amber-500 hover:text-amber-600 px-2 flex"
                >
                  Beacon block <ArrowTopRightOnSquareIcon className="w-6 h-6 ml-2" />
                </button>
              </dt>
            )}
            <dt className="text-md text-gray-500 font-bold">
              <a
                href={`/download/execution_payload_envelope/${id}`}
                download={
                  envelope
                    ? `execution_payload_envelope-${envelope.node}-${envelope.slot}-${envelope.block_root}.ssz`
                    : `execution_payload_envelope-${id}.ssz`
                }
                className="text-amber-500 hover:text-amber-600 px-2 flex"
              >
                Download <ArrowDownTrayIcon className="w-6 h-6 ml-2" />
              </a>
            </dt>
          </div>
        </dl>
      </div>
    </div>
  );
}
