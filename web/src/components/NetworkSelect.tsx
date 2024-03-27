import { Fragment, useMemo, useEffect } from 'react';

import { Listbox, Transition } from '@headlessui/react';
import { CheckIcon, ChevronUpDownIcon } from '@heroicons/react/20/solid';
import classNames from 'classnames';

import useNetwork from '@contexts/network';

const priorityArray = ['Mainnet', 'Holesky', 'Goerli', 'Sepolia'];

function reorderArray(inputArray: string[]): string[] {
  const orderedElements = inputArray
    .filter((item) => priorityArray.includes(item))
    .sort((a, b) => priorityArray.indexOf(a) - priorityArray.indexOf(b));

  const remainingElements = inputArray.filter((item) => !priorityArray.includes(item));

  return [...orderedElements, ...remainingElements];
}

export default function NetworkSelect({ networks: origNetworks }: { networks?: string[] }) {
  const networks = useMemo(() => reorderArray(origNetworks ?? []), []);
  const { network: currentNetwork, setNetwork } = useNetwork();

  useEffect(() => {
    if (!networks.includes(currentNetwork ?? '')) {
      setNetwork(networks[0]);
    }
  }, [networks, currentNetwork, setNetwork]);

  if (!currentNetwork) return null;

  return (
    <Listbox value={currentNetwork} onChange={setNetwork}>
      {({ open }) => (
        <div className="relative">
          <Listbox.Button className="pb-0.5 cursor-default rounded-md bg-transparent pl-3 pr-10 text-left text-sky-600 ring-inset focus:outline-none text-sm leading-6">
            <span className="block truncate underline font-extrabold">{currentNetwork}</span>
            <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
              <ChevronUpDownIcon className="h-5 w-5 text-sky-600" aria-hidden="true" />
            </span>
          </Listbox.Button>

          <Transition
            show={open}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="absolute z-10 max-h-60 w-full overflow-auto rounded-md bg-gray-200 py-1 shadow-lg focus:outline-none text-sm">
              {networks.map((network) => (
                <Listbox.Option
                  key={network}
                  className={({ focus }) =>
                    classNames(
                      focus ? 'bg-sky-600 text-white' : 'text-sky-600',
                      'relative cursor-default select-none py-1 pl-3 pr-9',
                    )
                  }
                  value={network}
                >
                  {({ selected, focus }) => (
                    <>
                      <span
                        className={classNames(
                          selected ? 'font-semibold' : 'font-normal',
                          'block truncate',
                        )}
                      >
                        {network}
                      </span>

                      {selected ? (
                        <span
                          className={classNames(
                            focus ? 'text-white' : 'text-sky-600',
                            'absolute inset-y-0 right-0 flex items-center pr-4',
                          )}
                        >
                          <CheckIcon
                            className={classNames(
                              focus ? 'text-amber-300' : 'text-amber-600',
                              'h-5 w-5',
                            )}
                            aria-hidden="true"
                          />
                        </span>
                      ) : null}
                    </>
                  )}
                </Listbox.Option>
              ))}
            </Listbox.Options>
          </Transition>
        </div>
      )}
    </Listbox>
  );
}
