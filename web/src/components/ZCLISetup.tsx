import { Disclosure } from '@headlessui/react';
import { InformationCircleIcon } from '@heroicons/react/20/solid';
import { ChevronDownIcon, ChevronUpIcon } from '@heroicons/react/24/outline';
import classNames from 'classnames';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { railscasts } from 'react-syntax-highlighter/dist/esm/styles/hljs';

import CopyToClipboard from '@components/CopyToClipboard';

export default function ZCLISetup() {
  const cmd = `# make sure go is installed
go install github.com/protolambda/zcli@latest

# check the command works
zcli --help`;
  return (
    <Disclosure>
      {({ open }) => (
        <>
          <Disclosure.Button
            className={classNames(
              open ? 'rounded-t-xl' : 'rounded-xl',
              'bg-white/25 mt-1 px-4 p-5 sm:px-6 min-w-full',
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
                      zcli
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
          </Disclosure.Button>

          <Disclosure.Panel className="text-gray-500 px-5 pb-5 bg-white/35 rounded-b-xl">
            <h3 className="text-base font-semibold leading-6 text-gray-600 pt-5">
              Install{' '}
              <a
                href="https://github.com/protolambda/zcli"
                target="_blank"
                className="text-sky-600 hover:text-sky-700 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
                rel="noreferrer"
              >
                zcli
              </a>
            </h3>
            <div className="mt-2">
              <div className="absolute right-10 sm:right-14 m-2 bg-white/35 mix-blend-hard-light hover:bg-white/20 rounded-lg cursor-pointer">
                <CopyToClipboard text={cmd} className="m-2" inverted />
              </div>
              <SyntaxHighlighter language="bash" style={railscasts} showLineNumbers wrapLines>
                {cmd}
              </SyntaxHighlighter>
            </div>
          </Disclosure.Panel>
        </>
      )}
    </Disclosure>
  );
}
