import { useMemo } from 'react';

import { ArrowDownTrayIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';

import { useBeaconStates } from '@hooks/useQuery';

export default function BeaconStateTable() {
  const { watch, setValue } = useFormContext();

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

  // fetchListBeaconState
  const { data, isLoading, error } = useBeaconStates({
    slot: beaconStateSlot ? parseInt(beaconStateSlot) : undefined,
    epoch: beaconStateEpoch ? parseInt(beaconStateEpoch) : undefined,
    state_root: beaconStateStateRoot ? beaconStateStateRoot : undefined,
    node: beaconStateNode ? beaconStateNode : undefined,
    node_version: beaconStateNodeVersion ? beaconStateNodeVersion : undefined,
    beacon_implementation: beaconStateNodeImplementation
      ? beaconStateNodeImplementation
      : undefined,
    pagination: {
      limit: 10,
      offset: 0,
      order_by: 'fetched_at DESC',
    },
  });

  const loading = useMemo(
    () =>
      Array.from({ length: 10 }, (_, i) => (
        <tr key={i} className="divide-x divide-orange-300">
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-48 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-16 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
            <div className="h-5 w-96 bg-gray-600/35 rounded-xl animate-pulse"></div>
          </td>
          <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600"></td>
        </tr>
      )),
    [],
  );

  let otherComp = undefined;

  // if loading or has error or has no data
  if (isLoading) {
    otherComp = loading;
  } else if (error) {
    let message = 'Something went wrong fetching filter values';
    if (typeof error === 'string') {
      message = error;
    }
    otherComp = <tr className="sm:col-span-6">{message}</tr>;
  } else if (!data || data.length === 0) {
    otherComp = <tr className="sm:col-span-6">No data available</tr>;
  }

  return (
    <table className="min-w-full divide-y divide-orange-500 sm:rounded-lg bg-white/55 shadow overflow-hidden">
      <thead className="bg-sky-400">
        <tr className="divide-x divide-orange-300">
          <th scope="col" className="py-3.5 pl-4 pr-4 text-left text-sm font-semibold text-gray-50">
            Fetched at
          </th>
          <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
            Node
          </th>
          <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
            Beacon node Implementation
          </th>
          <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
            Node version
          </th>
          <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
            Epoch
          </th>
          <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
            Slot
          </th>
          <th scope="col" className="px-4 py-3.5 text-left text-sm font-semibold text-gray-50">
            State root
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
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                  <span className="">{new Date(row.fetched_at).toISOString()}</span>
                </td>
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                  <span
                    className="cursor-pointer"
                    onClick={() => setValue('beaconStateNode', row.node)}
                  >
                    {row.node}
                  </span>
                </td>
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                  <span
                    className="cursor-pointer"
                    onClick={() =>
                      setValue('beaconStateNodeImplementation', row.beacon_implementation)
                    }
                  >
                    {row.beacon_implementation}
                  </span>
                </td>
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                  <span
                    className="cursor-pointer"
                    onClick={() => setValue('beaconStateNodeVersion', row.node_version)}
                  >
                    {row.node_version}
                  </span>
                </td>
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                  <span
                    className="cursor-pointer"
                    onClick={() => setValue('beaconStateEpoch', row.epoch)}
                  >
                    {row.epoch}
                  </span>
                </td>
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                  <span
                    className="cursor-pointer"
                    onClick={() => setValue('beaconStateSlot', row.slot)}
                  >
                    {row.slot}
                  </span>
                </td>
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600">
                  <span
                    className="cursor-pointer"
                    onClick={() => setValue('beaconStateStateRoot', row.state_root)}
                  >
                    {row.state_root}
                  </span>
                </td>
                <td className="whitespace-nowrap py-4 pl-4 pr-4 text-sm font-bold text-gray-600 flex flex-row">
                  <a href="#" className="text-sky-500 hover:text-sky-600 px-2">
                    <ArrowDownTrayIcon className="h-6 w-6" aria-hidden="true" />
                  </a>
                </td>
              </tr>
            ))}
      </tbody>
    </table>
  );
}
