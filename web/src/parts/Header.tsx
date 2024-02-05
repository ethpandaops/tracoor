import { Fragment, useState } from 'react';

import { Dialog, Transition } from '@headlessui/react';
import { InformationCircleIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { Link } from 'wouter';

import Logo from '@assets/logo.png';
import Walker from '@components/Walker';

export default function Header() {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <header className="bg-gray-200 shadow-2xl">
      <nav
        className="flex items-center justify-between px-6 py-3 lg:pl-8 lg:pr-14"
        aria-label="Global"
      >
        <div className="flex flex-1">
          <Link href="/" className="flex gap-2 items-center">
            <img src={Logo} className="object-contain w-12 h-12" />
            <span className="text-2xl underline decoration-sky-500/30 decoration-double underline-offset-4 drop-shadow-xl font-tracoor text-transparent bg-clip-text bg-gradient-to-r from-sky-600 via-amber-400 to-orange-500">
              Tracoor
            </span>
          </Link>
        </div>
        <div className="gap-5 flex">
          <button
            type="button"
            className=" inline-flex items-center justify-center rounded-md p-2 text-sky-600/70 transition hover:bg-gray-900/5"
            onClick={() => setMenuOpen(true)}
          >
            <span className="sr-only">Open menu</span>
            <InformationCircleIcon className="h-8 w-8" aria-hidden="true" />
          </button>
        </div>
      </nav>
      <Transition.Root show={menuOpen} as={Fragment}>
        <Dialog as="div" onClose={setMenuOpen}>
          <Transition.Child
            as={Fragment}
            enter="ease-in-out duration-100"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="ease-in-out duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-gray-300 bg-opacity-75 transition-opacity z-30" />
          </Transition.Child>
          <div className="fixed inset-0 overflow-hidden z-30">
            <div className="absolute inset-0 overflow-hidden">
              <div className="fixed inset-y-0 right-0 flex max-w-full pl-10">
                <Transition.Child
                  as={Fragment}
                  enter="transform transition ease-in-out duration-100 sm:duration-200"
                  enterFrom="translate-x-full"
                  enterTo="translate-x-0"
                  leave="transform transition ease-in-out duration-100 sm:duration-200"
                  leaveFrom="translate-x-0"
                  leaveTo="translate-x-full"
                >
                  <Dialog.Panel className="fixed inset-y-0 overflow-x-hidden right-0 w-full overflow-y-auto bg-gray-100 dark:bg-gray-900 sm:max-w-screen-lg sm:ring-1 sm:ring-white/10">
                    <div className="fixed opacity-10 pointer-events-none">
                      <Walker width={window.innerWidth < 1024 ? window.innerWidth : 1024} />
                    </div>
                    <div className="px-6 py-6">
                      <div className="flex flex-row-reverse">
                        <button
                          type="button"
                          className="mr-1.5 rounded-md p-1.5 text-gray-400 transition hover:bg-gray-900/5 dark:hover:bg-white/5"
                          onClick={() => setMenuOpen(false)}
                        >
                          <span className="sr-only">Close menu</span>
                          <XMarkIcon className="h-7 w-7" aria-hidden="true" />
                        </button>
                      </div>
                    </div>
                    <div className="relative pt-10 pb-20 sm:py-12">
                      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 relative">
                        <div className="mx-auto max-w-2xl lg:max-w-4xl lg:px-12">
                          <h1 className="font-display text-4xl font-bold tracking-tighter bg-clip-text text-transparent bg-gradient-to-r from-orange-400 via-blue-500 to-amber-600 dark:from-blue-200 dark:via-amber-200 dark:to-orange-300 sm:text-5xl lg:text-6xl">
                            An Ethereum beacon state and execution trace explorer
                          </h1>
                          <div className="mt-6 space-y-6 font-display text-xl sm:text-2xl tracking-tight text-gray-900 dark:text-gray-100">
                            <h4 className="font-display text-3xl font-bold tracking-tighter bg-clip-text">
                              About
                            </h4>
                            <p>
                              <span className="font-semibold">Tracoor</span> captures, stores and
                              makes available{' '}
                              <a
                                href="https://ethereum.github.io/beacon-APIs/#/Debug/getStateV2"
                                className="underline text-orange-500 hover:text-orange-600 dark:text-blue-200 dark:hover:text-blue-300"
                              >
                                beacon state
                              </a>
                              ,{' '}
                              <a
                                href="https://geth.ethereum.org/docs/interacting-with-geth/rpc/ns-debug#debugtraceblock"
                                className="underline text-orange-500 hover:text-orange-600 dark:text-blue-200 dark:hover:text-blue-300"
                              >
                                execution debug traces
                              </a>{' '}
                              and invalid gossiped verified blocks (see{' '}
                              <a
                                href="https://lighthouse-book.sigmaprime.io/help_bn.html"
                                className="underline text-orange-500 hover:text-orange-600 dark:text-blue-200 dark:hover:text-blue-300"
                              >
                                lighthouse
                              </a>{' '}
                              <span className="text-sm rounded-lg bg-gray-300 p-1">
                                --invalid-gossip-verified-blocks-path{' '}
                              </span>{' '}
                              flag).
                            </p>
                            <p>
                              While the <span className="font-semibold">Tracoor</span> source code
                              is maintained by the Ethereum Foundation DevOps team, instances can be
                              operated by the community ❤️
                            </p>
                            <p>
                              If you&apos;d like to run your own instance of{' '}
                              <span className="font-semibold">Tracoor</span>, checkout out the{' '}
                              <a
                                className="underline text-orange-500 hover:text-orange-600 dark:text-blue-200 dark:hover:text-blue-300"
                                href="https://github.com/ethpandaops/tracoor"
                              >
                                Github repository
                              </a>{' '}
                              for instructions.
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </Dialog.Panel>
                </Transition.Child>
              </div>
            </div>
          </div>
        </Dialog>
      </Transition.Root>
    </header>
  );
}
