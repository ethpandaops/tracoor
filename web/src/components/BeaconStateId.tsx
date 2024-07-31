import { ArrowDownTrayIcon, ArrowLeftStartOnRectangleIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import TimeAgo from 'react-timeago';
import { useLocation, Link } from 'wouter';

import Alert from '@components/Alert';
import CopyToClipboard from '@components/CopyToClipboard';
import useNetwork from '@contexts/network';
import { useBeaconStates } from '@hooks/useQuery';

export default function BeaconStateId({ id }: { id: string }) {
  const [, setLocation] = useLocation();
  const { setValue } = useFormContext();
  const { network } = useNetwork();
  const { data, isLoading, error } = useBeaconStates({
    network: network ? network : undefined,
    id: id,
    pagination: {
      limit: 1,
    },
  });

  const state = data?.[0];

  const handleSearch = (key: string, value: unknown) => {
    setValue(key, value);
    setLocation('/beacon_state');
  };

  let errorMessage = undefined;

  if (error) {
    errorMessage = 'Error fetching beacon states';
    if (error instanceof Error) {
      errorMessage = `Error fetching beacon states: ${error.message}`;
    }
  } else if (!isLoading && !state) {
    errorMessage = 'Beacon states not found';
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
                state && <TimeAgo date={new Date(state.fetched_at)} />
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Network</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              {isLoading ? (
                <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
              ) : (
                state?.network
              )}
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Node</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('beaconStateNode', state?.node)}
              >
                {isLoading ? (
                  <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  state?.node
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
                  handleSearch('beaconStateNodeImplementation', state?.beacon_implementation)
                }
              >
                {isLoading ? (
                  <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  state?.beacon_implementation
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
                onClick={() => handleSearch('beaconStateNodeVersion', state?.node_version)}
              >
                {isLoading ? (
                  <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  state?.node_version
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
                onClick={() => handleSearch('beaconStateEpoch', state?.epoch.toString())}
              >
                {isLoading ? (
                  <div className="h-5 w-20 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  state?.epoch
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
              <CopyToClipboard
                text={state?.epoch.toString() ?? ''}
                className="ml-2 hidden lg:block"
              />
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">Slot</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4 flex">
              <span
                className="relative top-1 group transition cursor-pointer"
                onClick={() => handleSearch('beaconStateSlot', state?.slot.toString())}
              >
                {isLoading ? (
                  <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
                ) : (
                  state?.slot
                )}
                <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
              </span>
              <CopyToClipboard
                text={state?.slot.toString() ?? ''}
                className="ml-2 hidden lg:block"
              />
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:grid sm:grid-cols-5 sm:gap-4 sm:px-6">
            <dt className="text-sm font-medium text-gray-500">State root</dt>
            <dd className="mt-1 text-sm text-sky-500 font-bold sm:mt-0 sm:col-span-4">
              <span className="lg:hidden font-mono flex">
                <span
                  className="relative top-1 group transition duration-300 cursor-pointer"
                  onClick={() => handleSearch('beaconStateStateRoot', state?.state_root)}
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    state?.state_root
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-400"></span>
                </span>
              </span>
              <span className="hidden lg:flex font-mono">
                <span
                  className="relative top-1 group transition cursor-pointer"
                  onClick={() => handleSearch('beaconStateStateRoot', state?.state_root)}
                >
                  {isLoading ? (
                    <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
                  ) : (
                    state?.state_root
                  )}
                  <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-amber-300"></span>
                </span>
                <CopyToClipboard text={state?.state_root ?? ''} className="ml-2" />
              </span>
            </dd>
          </div>
          <div className="py-4 sm:py-5 px-4 sm:px-6 flex justify-center sm:bg-gray-100 sm:dark:bg-gray-900">
            <dt className="text-md text-gray-500 font-bold">
              <a
                href={`/download/beacon_state/${id}`}
                download={
                  state
                    ? `beacon_state-${state.node}-${state.slot}-${state.state_root}.ssz`
                    : `beacon_state-${id}.ssz`
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
              setValue('beaconStateSelectorId', state?.id);
              setValue('beaconBlockSelectorSlot', '');
              setLocation(
                `/lcli_state_transition?beaconStateSelectorId=${state?.id}&beaconBlockSelectorSlot=${state?.slot}`,
              );
            }}
            href={`/lcli_state_transition?beaconStateSelectorId=${state?.id}&beaconBlockSelectorSlot=${state?.slot}`}
            className="py-4 sm:py-5 px-4 sm:px-6 flex text-gray-100 font-bold justify-center sm:justify-start bg-amber-500/85"
          >
            <ArrowLeftStartOnRectangleIcon className="w-6 h-6 mr-2" /> lcli state transition
          </Link>
          <Link
            onClick={(a) => {
              a.preventDefault();
              setValue('beaconStateSelectorId', state?.id);
              setValue('beaconBlockSelectorSlot', '');
              setLocation(
                `/ncli_state_transition?beaconStateSelectorId=${state?.id}&beaconBlockSelectorSlot=${state?.slot}`,
              );
            }}
            href={`/ncli_state_transition?beaconStateSelectorId=${state?.id}&beaconBlockSelectorSlot=${state?.slot}`}
            className="py-4 sm:py-5 px-4 sm:px-6 flex text-gray-100 font-bold justify-center sm:justify-start bg-amber-500/85"
          >
            <ArrowLeftStartOnRectangleIcon className="w-6 h-6 mr-2" /> ncli state transition
          </Link>
          <Link
            onClick={(a) => {
              a.preventDefault();
              setValue('beaconStateSelectorId', state?.id);
              setLocation(`/zcli_state_diff?beaconStateSelectorId=${state?.id}`);
            }}
            href={`/zcli_state_diff?beaconStateSelectorId=${state?.id}`}
            className="py-4 sm:py-5 px-4 sm:px-6 flex text-gray-100 font-bold justify-center sm:justify-start bg-amber-500/85"
          >
            <ArrowLeftStartOnRectangleIcon className="w-6 h-6 mr-2" /> zcli state diff
          </Link>
        </dl>
      </div>
    </div>
  );
}
