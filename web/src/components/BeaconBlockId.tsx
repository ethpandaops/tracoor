import { ArrowDownTrayIcon, ArrowLeftStartOnRectangleIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import TimeAgo from 'react-timeago';
import { useLocation, Link } from 'wouter';

import Alert from '@components/Alert';
import CopyToClipboard from '@components/CopyToClipboard';
import useNetwork from '@contexts/network';
import { useBeaconBlocks } from '@hooks/useQuery';

export default function BeaconBlockId({ id }: { id: string }) {
  const [, setLocation] = useLocation();
  const { setValue } = useFormContext();
  const { network } = useNetwork();
  const { data, isLoading, error } = useBeaconBlocks({
    network: network ? network : undefined,
    id: id,
    pagination: {
      limit: 1,
    },
  });

  const block = data?.[0];

  const handleSearch = (key: string, value: unknown) => {
    setValue(key, value);
    setLocation('/beacon_block');
  };

  let errorMessage = undefined;

  if (error) {
    errorMessage = 'Error fetching beacon block';
    if (error instanceof Error) {
      errorMessage = `Error fetching beacon block: ${error.message}`;
    }
  } else if (!isLoading && !block) {
    errorMessage = 'Beacon block not found';
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
                block && <TimeAgo date={new Date(block.fetched_at)} />
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Network</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              {isLoading ? (
                <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
              ) : (
                block?.network
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Node</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('beaconBlockNode', block?.node)}
              >
                {isLoading ? (
                  <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  block?.node
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
                  handleSearch('beaconBlockNodeImplementation', block?.beacon_implementation)
                }
              >
                {isLoading ? (
                  <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  block?.beacon_implementation
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
                onClick={() => handleSearch('beaconBlockNodeVersion', block?.node_version)}
              >
                {isLoading ? (
                  <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  block?.node_version
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
                onClick={() => handleSearch('beaconBlockEpoch', block?.epoch.toString())}
              >
                {isLoading ? (
                  <div className="h-5 w-20 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  block?.epoch
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
              <CopyToClipboard
                text={block?.epoch.toString() ?? ''}
                className="ml-2 hidden lg:block"
              />
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Slot</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('beaconBlockSlot', block?.slot.toString())}
              >
                {isLoading ? (
                  <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  block?.slot
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
              <CopyToClipboard
                text={block?.slot.toString() ?? ''}
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
                  onClick={() => handleSearch('beaconBlockBlockRoot', block?.block_root)}
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    block?.block_root
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-400"></span>
                </span>
              </span>
              <span className="hidden lg:flex font-mono">
                <span
                  className="relative top-1 group transition cursor-pointer"
                  onClick={() => handleSearch('beaconBlockBlockRoot', block?.block_root)}
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    block?.block_root
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
                </span>
                <CopyToClipboard text={block?.block_root ?? ''} className="ml-2" />
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:px-6 flex justify-center sm:bg-gray-100 sm:dark:bg-gray-900">
            <dt className="text-md text-gray-500 font-bold">
              <a
                href={`/download/beacon_block/${id}`}
                download={
                  block
                    ? `beacon_block-${block.node}-${block.slot}-${block.block_root}.ssz`
                    : `beacon_block-${id}.ssz`
                }
                className="text-amber-500 hover:text-amber-600 px-2 flex"
              >
                Download <ArrowDownTrayIcon className="w-6 h-6 ml-2" />
              </a>
            </dt>
          </div>
          <Link
            onClick={(a) => {
              a.preventDefault();
              setValue('beaconBlockSelectorId', block?.id);
              setValue('beaconStateSelectorSlot', '');
              setLocation(
                `/lcli_state_transition?beaconBlockSelectorId=${block?.id}&beaconStateSelectorSlot=${block?.slot}`,
              );
            }}
            href={`/lcli_state_transition?beaconBlockSelectorId=${block?.id}&beaconStateSelectorSlot=${block?.slot}`}
            className="py-4 sm:py-5 px-4 sm:px-6 flex text-gray-100 font-bold justify-center sm:justify-start bg-amber-500/85"
          >
            <ArrowLeftStartOnRectangleIcon className="w-6 h-6 mr-2" /> lcli state transition
          </Link>
          <Link
            onClick={(a) => {
              a.preventDefault();
              setValue('beaconBlockSelectorId', block?.id);
              setValue('beaconStateSelectorSlot', '');
              setLocation(
                `/ncli_state_transition?beaconBlockSelectorId=${block?.id}&beaconStateSelectorSlot=${block?.slot}`,
              );
            }}
            href={`/ncli_state_transition?beaconBlockSelectorId=${block?.id}&beaconStateSelectorSlot=${block?.slot}`}
            className="py-4 sm:py-5 px-4 sm:px-6 flex text-gray-100 font-bold justify-center sm:justify-start bg-amber-500/85"
          >
            <ArrowLeftStartOnRectangleIcon className="w-6 h-6 mr-2" /> ncli state transition
          </Link>
          <Link
            onClick={(a) => {
              a.preventDefault();
              setValue('beaconStateSelectorId', block?.id);
              setLocation(`/zcli_state_diff?beaconStateSelectorId=${block?.id}`);
            }}
            href={`/zcli_state_diff?beaconStateSelectorId=${block?.id}`}
            className="py-4 sm:py-5 px-4 sm:px-6 flex text-gray-100 font-bold justify-center sm:justify-start bg-amber-500/85"
          >
            <ArrowLeftStartOnRectangleIcon className="w-6 h-6 mr-2" /> zcli state diff
          </Link>
        </dl>
      </div>
    </div>
  );
}
