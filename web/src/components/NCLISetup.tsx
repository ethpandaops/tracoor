import { Disclosure, DisclosureButton, DisclosurePanel } from '@headlessui/react';
import { InformationCircleIcon } from '@heroicons/react/20/solid';
import { ChevronDownIcon, ChevronUpIcon } from '@heroicons/react/24/outline';
import classNames from 'classnames';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { railscasts } from 'react-syntax-highlighter/dist/esm/styles/hljs';

import CopyToClipboard from '@components/CopyToClipboard';
import useNetwork from '@contexts/network';
import { useConfig } from '@hooks/useQuery';
import { getNCLIConfig, isCustomNetwork, getNetworkConfig } from '@utils/config';

export default function NCLISetup() {
  const { data: config } = useConfig({});
  const { network } = useNetwork();

  const ncliConfig = getNCLIConfig(config ?? {});
  const devnetConfig = getNetworkConfig(config ?? {});

  const cmd = `# download the source code
git clone https://github.com/${ncliConfig.repository}.git
cd nimbus-eth2
${
  ncliConfig.branch
    ? `
# checkout the branch
git checkout ${ncliConfig.branch}
`
    : ''
}
# build ncli
make NIMFLAGS="-d:release" ncli

${
  isCustomNetwork(config ?? {})
    ? `# pull the network config locally
TMP_DIR="$(mktemp -d)" && git clone --depth 1 --branch ${devnetConfig.branch} https://github.com/${devnetConfig.repository}.git $TMP_DIR && cp -r $TMP_DIR/${devnetConfig.path} ./${network} && rm -rf $TMP_DIR

`
    : ''
}# check the command works
build/ncli --help`;
  return (
    <Disclosure>
      {({ open }) => (
        <>
          <DisclosureButton
            className={classNames(
              open ? 'rounded-t-xl border-t-2 border-x-2' : 'rounded-xl border-2',
              'bg-white/25 mt-1 px-4 p-5 sm:px-6 min-w-full border-amber-200',
            )}
          >
            <h3 className="text-base font-semibold leading-6 text-gray-700 flex justify-between items-center">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <InformationCircleIcon className="h-8 w-8 text-sky-600" aria-hidden="true" />
                </div>
                <div className="ml-3 flex-1 md:flex md:justify-between">
                  <h2 className="text-lg text-sky-700">
                    How to setup{' '}
                    <span className="bg-white/35 rounded-lg font-mono px-2 py-1 text-gray-600">
                      ncli
                    </span>
                  </h2>
                </div>
              </div>
              <span>
                {open ? (
                  <ChevronUpIcon className="h-8 w-8 ml-2 text-gray-500" aria-hidden="true" />
                ) : (
                  <ChevronDownIcon className="h-8 w-8 ml-2 text-gray-500" aria-hidden="true" />
                )}
              </span>
            </h3>
          </DisclosureButton>

          <DisclosurePanel className="text-gray-500 px-5 pb-5 bg-white/35 rounded-b-xl border-b-2 border-x-2 border-amber-200">
            <h3 className="text-base font-semibold leading-6 text-gray-600 pt-5">
              Install{' '}
              <a
                href="https://github.com/status-im/nimbus-eth2/tree/stable/ncli"
                target="_blank"
                className="text-sky-600 hover:text-sky-700 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
                rel="noreferrer"
              >
                ncli
              </a>
            </h3>
            <div className="mt-2">
              <div className="absolute right-12 sm:right-16 m-2 bg-white/35 mix-blend-hard-light hover:bg-white/20 rounded-lg cursor-pointer">
                <CopyToClipboard text={cmd} className="m-2" inverted />
              </div>
              <div className="border-2 border-gray-200">
                <SyntaxHighlighter language="bash" style={railscasts} showLineNumbers wrapLines>
                  {cmd}
                </SyntaxHighlighter>
              </div>
            </div>
          </DisclosurePanel>
        </>
      )}
    </Disclosure>
  );
}
