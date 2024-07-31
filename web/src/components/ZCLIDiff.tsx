import { useMemo } from 'react';

import { XMarkIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { railscasts } from 'react-syntax-highlighter/dist/esm/styles/hljs';

import { BeaconState } from '@app/types/api';
import Alert from '@components/Alert';
import BeaconStateSelector from '@components/BeaconStateSelector';
import CopyToClipboard from '@components/CopyToClipboard';
import Loading from '@components/Loading';
import ZCLISetup from '@components/ZCLISetup';
import useNetwork from '@contexts/network';
import { useBeaconStates } from '@hooks/useQuery';

export default function ZCLIStateTransition() {
  const { register, watch, setValue } = useFormContext();
  const { network } = useNetwork();

  const [zcliFileName, beaconStateSelectorId] = watch(['zcliFileName', 'beaconStateSelectorId']);

  const {
    data: stateData,
    isLoading: stateIsLoading,
    error: stateError,
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

  function generateStateFileNamePrefix(state: BeaconState) {
    return `beacon_state-${state.node}-${state.slot}-${state.state_root}`;
  }

  const state = stateData?.[0];
  let stateFileName = '';
  if (state) {
    stateFileName = generateStateFileNamePrefix(state);
  }

  const cmd = useMemo(() => {
    if (state && zcliFileName) {
      return `# Download the state
# Note: requires wget
wget -O ${stateFileName}.ssz -q ${window.location.origin}/download/beacon_state/${state.id}

# Diff the states
# Note: change "deneb" to the correct phase as required
zcli \\
  diff \\
  deneb \\
  BeaconState \\
  ssz:${stateFileName}.ssz \\
  ssz:${zcliFileName}`;
    }
    return '';
  }, [stateData, zcliFileName]);

  let otherComp = undefined;

  if (stateIsLoading) {
    otherComp = <Loading />;
  } else if (stateError) {
    let message = 'Something went wrong fetching data';
    if (typeof stateError === 'string') {
      message = stateError;
    }
    otherComp = <Alert type="error" message={message} />;
  } else if (cmd && !state) {
    otherComp = <Alert type="error" message="Beacon state data not found" />;
  }

  return (
    <div className="mx-2 mt-8">
      <ZCLISetup />
      <BeaconStateSelector />
      <div className="bg-white/35 my-10 px-8 py-5 rounded-xl border-2 border-amber-200">
        <div className="absolute -mt-8 bg-white px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold border-2 border-sky-400">
          Local filename
        </div>
        {zcliFileName && (
          <button
            className="absolute right-8 sm:right-14 -mt-8 bg-white px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-gray-600 font-bold flex cursor-pointer transition hover:text-gray-800 border-2 border-gray-500 hover:border-gray-700"
            onClick={() => setValue(`zcliFileName`, '')}
          >
            Clear
            <XMarkIcon className="w-4 h-4" />
          </button>
        )}
        <h3 className="text-lg font-bold my-5 text-gray-700">
          Type the local filename of the state to diff
        </h3>
        <div className="bg-white/35 border-lg rounded-lg p-4 border-2 border-amber-100">
          <label htmlFor="zcliFileName" className="block text-sm font-bold leading-6 text-gray-700">
            Filename
          </label>
          <div className="mt-2">
            <input
              {...register('zcliFileName')}
              type="text"
              className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
            />
          </div>
        </div>
      </div>
      {(otherComp || cmd) && (
        <div className="bg-white/35 my-10 px-8 py-5 rounded-xl border-2 border-amber-200">
          <div className="absolute -mt-8 bg-white px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold">
            State diff command
          </div>
          <div className="mt-2">
            {otherComp}
            {!otherComp && (
              <div className="border-2 border-gray-200">
                <div className="absolute right-14 sm:right-20 m-2 bg-white/35 mix-blend-hard-light hover:bg-white/20 rounded-lg cursor-pointer">
                  <CopyToClipboard text={cmd} className="m-2" inverted />
                </div>
                <SyntaxHighlighter language="bash" style={railscasts} showLineNumbers wrapLines>
                  {cmd}
                </SyntaxHighlighter>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
