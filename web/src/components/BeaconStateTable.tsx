import { useState, useMemo, Fragment } from 'react';

import { Dialog, Transition } from '@headlessui/react';
import {
  ArrowDownTrayIcon,
  ArrowUpIcon,
  ArrowDownIcon,
  ArrowsUpDownIcon,
  XMarkIcon,
  MagnifyingGlassCircleIcon,
} from '@heroicons/react/24/outline';
import classNames from 'classnames';
import { useFormContext } from 'react-hook-form';
import TimeAgo from 'react-timeago';
import { Link, useLocation } from 'wouter';

import BeaconStateId from '@components/BeaconStateId';
import Pagination from '@components/Pagination';
import useNetwork from '@contexts/network';
import { Selection } from '@contexts/selection';
import { useBeaconStates, useBeaconStatesCount } from '@hooks/useQuery';

type SortConfig = {
  key: string;
  direction: 'ASC' | 'DESC';
};

export default function BeaconStateTable({ id }: { id?: string }) {
  const { network } = useNetwork();
  const [, setLocation] = useLocation();
  const [sortConfig, setSortConfig] = useState<SortConfig>({
    key: 'fetched_at',
    direction: 'DESC',
  });
  const { watch, setValue } = useFormContext();
  const [currentPage, setCurrentPage] = useState<number>(1);
  const itemsPerPage = 100;

  const handleSort = (key: string) => {
    setSortConfig((currentSortConfig: SortConfig) => {
      if (currentSortConfig.key === key) {
        return {
          key,
          direction: currentSortConfig.direction === 'ASC' ? 'DESC' : 'ASC',
        };
      }
      return { key, direction: 'DESC' };
    });
  };

  const [
    beaconStateSlot,
    beaconStateEpoch,
    beaconStateStateRoot,
    beaconStateNode,
    beaconStateNodeImplementation,
    beaconStateNodeVersion,
  ] = watch([
    'beaconStateSlot',
    'beaconStateEpoch',
    'beaconStateStateRoot',
    'beaconStateNode',
    'beaconStateNodeImplementation',
    'beaconStateNodeVersion',
  ]);

  const { data, isLoading, error } = useBeaconStates({
    network: network ? network : undefined,
    slot: beaconStateSlot ? parseInt(beaconStateSlot) : undefined,
    epoch: beaconStateEpoch ? parseInt(beaconStateEpoch) : undefined,
    state_root: beaconStateStateRoot ? beaconStateStateRoot : undefined,
    node: beaconStateNode ? beaconStateNode : undefined,
    node_version: beaconStateNodeVersion ? beaconStateNodeVersion : undefined,
    beacon_implementation: beaconStateNodeImplementation
      ? beaconStateNodeImplementation
      : undefined,
    pagination: {
      limit: itemsPerPage,
      offset: (currentPage - 1) * itemsPerPage,
      order_by: `${sortConfig.key} ${sortConfig.direction}`,
    },
  });

  const { data: count } = useBeaconStatesCount({
    network: network ? network : undefined,
    slot: beaconStateSlot ? parseInt(beaconStateSlot) : undefined,
    epoch: beaconStateEpoch ? parseInt(beaconStateEpoch) : undefined,
    state_root: beaconStateStateRoot ? beaconStateStateRoot : undefined,
    node: beaconStateNode ? beaconStateNode : undefined,
    node_version: beaconStateNodeVersion ? beaconStateNodeVersion : undefined,
    beacon_implementation: beaconStateNodeImplementation
      ? beaconStateNodeImplementation
      : undefined,
  });

  const totalPages = count ? Math.ceil(count / itemsPerPage) : 0;

  const loading = useMemo(
    () =>
      Array.from({ length: itemsPerPage }, (_, i) => (
        <tr key={i} className="divide-x divide-orange-300">
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden md:table-cell">
            <div className="h-5 w-28 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-64 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 2xl:table-cell">
            <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 4xl:table-cell">
            <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 3xl:table-cell">
            <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-24 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 2xl:table-cell">
            <div className="h-5 w-[550px] bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap w-0 py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-20 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
        </tr>
      )),
    [],
  );

  let otherComp = undefined;

  if (isLoading) {
    otherComp = loading;
  } else if (error) {
    let message = 'Something went wrong fetching data';
    if (typeof error === 'string') {
      message = error;
    }
    otherComp = (
      <tr className="">
        <td
          colSpan={8}
          className="whitespace-nowrap py-4 pl-4 pr-4 font-bold text-red-600 text-center text-xl"
        >
          {message}
        </td>
      </tr>
    );
  } else if (!data || data.length === 0) {
    otherComp = (
      <tr className="">
        <td
          colSpan={8}
          className="whitespace-nowrap py-4 pl-4 pr-4 font-bold text-gray-600 text-center text-xl"
        >
          No data available
        </td>
      </tr>
    );
  }

  return (
    <>
      <div className="mt-8 flow-root">
        <div className="-mx-4 -my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
          <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
            <table className="min-w-full divide-y divide-orange-500 sm:rounded-lg bg-white/55 shadow overflow-hidden">
              <thead className="bg-sky-400">
                <tr className="divide-x divide-orange-300">
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'fetched_at' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'fetched_at' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'fetched_at' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden md:table-cell',
                    )}
                    onClick={() => handleSort('fetched_at')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Fetched at</span>
                      <span>
                        {sortConfig.key === 'fetched_at' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'node' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'node' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'node' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0',
                    )}
                    onClick={() => handleSort('node')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Node</span>
                      <span>
                        {sortConfig.key === 'node' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'beacon_implementation' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'beacon_implementation' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'beacon_implementation' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden 2xl:table-cell',
                    )}
                    onClick={() => handleSort('beacon_implementation')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Beacon node Implementation</span>
                      <span>
                        {sortConfig.key === 'beacon_implementation' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'node_version' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'node_version' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'node_version' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden 4xl:table-cell',
                    )}
                    onClick={() => handleSort('node_version')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Node version</span>
                      <span>
                        {sortConfig.key === 'node_version' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'epoch' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'epoch' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'epoch' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0 hidden 3xl:table-cell',
                    )}
                    onClick={() => handleSort('epoch')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Epoch</span>
                      <span>
                        {sortConfig.key === 'epoch' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'slot' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'slot' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'slot' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 w-0',
                    )}
                    onClick={() => handleSort('slot')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">Slot</span>
                      <span>
                        {sortConfig.key === 'slot' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className={classNames(
                      sortConfig.key !== 'state_root' ? 'cursor-s-resize' : '',
                      sortConfig.key === 'state_root' && sortConfig.direction === 'DESC'
                        ? 'cursor-n-resize'
                        : '',
                      sortConfig.key === 'state_root' && sortConfig.direction === 'ASC'
                        ? 'cursor-s-resize'
                        : '',
                      'py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50 hidden 2xl:table-cell',
                    )}
                    onClick={() => handleSort('state_root')}
                  >
                    <div className="flex">
                      <span className="whitespace-nowrap">State root</span>
                      <span>
                        {sortConfig.key === 'state_root' ? (
                          sortConfig.direction === 'DESC' ? (
                            <ArrowDownIcon className="ml-2 h-5 w-5" />
                          ) : (
                            <ArrowUpIcon className="ml-2 h-5 w-5" />
                          )
                        ) : (
                          <ArrowsUpDownIcon className="ml-2 h-5 w-5" />
                        )}
                      </span>
                    </div>
                  </th>
                  <th
                    scope="col"
                    className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50"
                  ></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-orange-300">
                {otherComp
                  ? otherComp
                  : data?.map((row) => (
                      <tr key={row.id} className="divide-x divide-orange-300">
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-0 hidden md:table-cell">
                          <span className="underline decoration-dotted underline-offset-2 cursor-help">
                            <TimeAgo date={new Date(row.fetched_at)} />
                          </span>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                          <div className=" w-fit">
                            <span
                              className="relative top-1 group transition cursor-pointer"
                              onClick={() => setValue('beaconStateNode', row.node)}
                            >
                              {row.node}
                              <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                            </span>
                          </div>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 2xl:table-cell">
                          <div className=" w-fit">
                            <span
                              className="relative top-1 group transition cursor-pointer"
                              onClick={() =>
                                setValue('beaconStateNodeImplementation', row.beacon_implementation)
                              }
                            >
                              {row.beacon_implementation}
                              <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                            </span>
                          </div>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 4xl:table-cell">
                          <div className=" w-fit">
                            <span
                              className="relative top-1 group transition cursor-pointer"
                              onClick={() => setValue('beaconStateNodeVersion', row.node_version)}
                            >
                              {row.node_version}
                              <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                            </span>
                          </div>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 3xl:table-cell">
                          <div className=" w-fit">
                            <span
                              className="relative top-1 group transition cursor-pointer"
                              onClick={() => setValue('beaconStateEpoch', row.epoch)}
                            >
                              {row.epoch}
                              <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                            </span>
                          </div>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                          <div className=" w-fit">
                            <span
                              className="relative top-1 group transition cursor-pointer"
                              onClick={() => setValue('beaconStateSlot', row.slot)}
                            >
                              {row.slot}
                              <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                            </span>
                          </div>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 hidden 2xl:table-cell">
                          <div className=" w-fit">
                            <span
                              className="relative top-1 group transition cursor-pointer font-mono"
                              onClick={() => setValue('beaconStateStateRoot', row.state_root)}
                            >
                              {row.state_root}
                              <span className="relative -top-0.5 block max-w-0 group-hover:max-w-full transition-all duration-500 h-0.5 bg-sky-400"></span>
                            </span>
                          </div>
                        </td>
                        <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 w-1">
                          <div className="flex flex-row ">
                            <a
                              href={`/download/beacon_state/${row.id}`}
                              download={`beacon_state-${row.node}-${row.slot}-${row.state_root}.ssz.gz`}
                              className="text-sky-500 hover:text-sky-600 px-2"
                            >
                              <ArrowDownTrayIcon className="h-6 w-6" aria-hidden="true" />
                            </a>
                            <Link href={`/beacon_state/${row.id}`}>
                              <span className="text-sky-500 hover:text-sky-600 px-2 cursor-pointer">
                                <MagnifyingGlassCircleIcon className="h-6 w-6" aria-hidden="true" />
                              </span>
                            </Link>
                          </div>
                        </td>
                      </tr>
                    ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <div className={classNames(totalPages <= 1 ? 'hidden' : '', 'mt-10 mb-20')}>
        <Pagination
          currentPage={currentPage}
          totalPages={totalPages}
          onPageChange={(page: number) => setCurrentPage(page)}
        />
      </div>

      <Transition.Root show={Boolean(id)} as={Fragment}>
        <Dialog as="div" onClose={() => setLocation(`/${Selection.beacon_state}`)}>
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
                  <Dialog.Panel className="fixed inset-y-0 overflow-x-hidden right-0 w-full overflow-y-auto bg-gray-100 sm:ring-1 sm:ring-white/10 sm:max-w-screen-lg">
                    <div className="flex h-full flex-col py-6 shadow-xl">
                      <div className="px-4 mb-6 mt-1 sm:px-6">
                        <div className="flex items-start justify-between">
                          <Dialog.Title className="mt-1 flex items-center text-base font-semibold leading-6 text-amber-600">
                            Beacon State
                          </Dialog.Title>
                          <div className="ml-3 flex h-7 items-center">
                            <button
                              type="button"
                              className="rounded-md p-1.5 text-gray-400 transition hover:bg-gray-900/5"
                              onClick={() => setLocation(`/${Selection.beacon_state}`)}
                            >
                              <span className="sr-only">Close menu</span>
                              <XMarkIcon className="h-7 w-7" aria-hidden="true" />
                            </button>
                          </div>
                        </div>
                      </div>
                      {id && <BeaconStateId id={id} />}
                    </div>
                  </Dialog.Panel>
                </Transition.Child>
              </div>
            </div>
          </div>
        </Dialog>
      </Transition.Root>
    </>
  );
}
