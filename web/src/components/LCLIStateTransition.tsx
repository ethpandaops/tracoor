import { useMemo } from 'react';

import { ArrowRightIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { railscasts } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { Link, useLocation } from 'wouter';

import { BeaconBlock, BeaconState, BeaconBadBlock } from '@app/types/api';
import Alert from '@components/Alert';
import BeaconBlockSelector from '@components/BeaconBlockSelector';
import BeaconStateSelector from '@components/BeaconStateSelector';
import CopyToClipboard from '@components/CopyToClipboard';
import LCLISetup from '@components/LCLISetup';
import Loading from '@components/Loading';
import useNetwork from '@contexts/network';
import { useBeaconBlocks, useBeaconBadBlocks, useBeaconStates } from '@hooks/useQuery';

export default function LCLIStateTransition() {
  const { watch, setValue } = useFormContext();
  const [, setLocation] = useLocation();
  const { network } = useNetwork();

  const [beaconBlockSelectorId, beaconStateSelectorId] = watch([
    'beaconBlockSelectorId',
    'beaconStateSelectorId',
  ]);

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

  const {
    data: blockData,
    isLoading: blockIsLoading,
    error: blockError,
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
    data: badBlockData,
    isLoading: badBlockIsLoading,
    error: badBlockError,
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

  function generateStateFileNamePrefix(state: BeaconState) {
    return `beacon_state-${state.node}-${state.slot}-${state.state_root}`;
  }

  function generateBlockFileNamePrefix(block: BeaconBlock) {
    return `beacon_block-${block.node}-${block.slot}-${block.block_root}`;
  }

  function generateBadBlockFileNamePrefix(block: BeaconBadBlock) {
    return `beacon_bad_block-${block.node}-${block.slot}-${block.block_root}`;
  }

  const state = stateData?.[0];
  const block = blockData?.[0];
  const badBlock = badBlockData?.[0];
  let stateFileName = '';
  let blockFileName = '';
  let blockType = 'beacon_block';
  if (state) {
    stateFileName = generateStateFileNamePrefix(state);
  }
  if (block) {
    blockFileName = generateBlockFileNamePrefix(block);
  }
  if (badBlock) {
    blockFileName = generateBadBlockFileNamePrefix(badBlock);
    blockType = 'beacon_bad_block';
  }

  const cmd = useMemo(() => {
    if (state && (block || badBlock)) {
      return `# Download the state and block
# Note: requires wget
wget -O ${stateFileName}.ssz -q ${window.location.origin}/download/beacon_state/${state.id}
wget -O ${blockFileName}.ssz -q ${window.location.origin}/download/${blockType}/${block?.id ?? badBlock?.id}


# Transition the state
cargo run --release -- transition-blocks \\
  --network ${network} \\
  --pre-state-path ${stateFileName}.ssz \\
  --block-path ${blockFileName}.ssz \\
  --post-state-output-path ${stateFileName}-post.ssz`;
    }
    return '';
  }, [stateData, blockData, badBlockData]);

  let otherComp = undefined;

  if (stateIsLoading || blockIsLoading || badBlockIsLoading) {
    otherComp = <Loading />;
  } else if (stateError || blockError || badBlockError) {
    let message = 'Something went wrong fetching data';
    if (typeof stateError === 'string') {
      message = stateError;
    }
    if (typeof blockError === 'string') {
      message = blockError;
    }
    if (typeof badBlockError === 'string') {
      message = badBlockError;
    }
    otherComp = <Alert type="error" message={message} />;
  } else if (cmd && !state) {
    otherComp = <Alert type="error" message="Beacon state data not found" />;
  } else if (cmd && !block && !badBlock) {
    otherComp = <Alert type="error" message="Beacon block data not found" />;
  }

  return (
    <div className="mx-2 mt-8">
      <LCLISetup />
      <BeaconStateSelector />
      <BeaconBlockSelector />
      {(otherComp || cmd) && (
        <>
          <div className="bg-white/35 my-10 px-8 py-5 rounded-xl border-2 border-amber-200">
            <div className="absolute -mt-8 bg-white px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold border-2 border-sky-400">
              State Transition command
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
          {!otherComp && (
            <div className="rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50 flex items-center justify-between">
              <h3 className="text-base font-semibold leading-6">
                You can use{' '}
                <a
                  href="https://github.com/protolambda/zcli"
                  target="_blank"
                  className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
                  rel="noreferrer"
                >
                  zcli
                </a>{' '}
                to get the state transition difference between this and an existing beacon state in
                tracoor
              </h3>
              <Link
                className="bg-white/15 rounded-xl px-2 py-2 border-2 border-amber-400 ml-4 hover:border-amber-300 flex items-center justify-center whitespace-nowrap"
                onClick={(a) => {
                  a.preventDefault();
                  setValue('zcliFileName', `${stateFileName}-post.ssz`);
                  setValue('beaconStateSelectorId', '');
                  setValue('beaconStateSelectorSlot', block?.slot);
                  setLocation(
                    `/zcli_state_diff?zcliFileName=${stateFileName}-post.ssz&beaconStateSelectorSlot=${block?.slot}`,
                  );
                }}
                href={`/zcli_state_diff?zcliFileName=${stateFileName}-post.ssz&beaconStateSelectorSlot=${block?.slot}`}
              >
                Lets go <ArrowRightIcon className="ml-2 h-5 w-5" />
              </Link>
            </div>
          )}
        </>
      )}
    </div>
  );
}
