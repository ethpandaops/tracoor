import { useFormContext, Controller } from 'react-hook-form';

import Alert from '@components/Alert';
import { CustomCombobox } from '@components/Combobox';
import DebouncedInput from '@components/DebouncedInput';
import { useUniqueBeaconBadBlobValues } from '@hooks/useQuery';

export default function FilterForm() {
  const { control } = useFormContext();

  const { data, isLoading, error } = useUniqueBeaconBadBlobValues([
    'node',
    'node_version',
    'beacon_implementation',
  ]);

  let errorComp = undefined;

  if (!isLoading && error) {
    let message = 'Something went wrong fetching filter values';
    if (typeof error === 'string') {
      message = error;
    }
    errorComp = <Alert type="error" message={message} />;
  }

  return (
    <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-10">
      <div className="sm:col-span-2 sm:col-start-1">
        <label
          htmlFor="beaconBadBlobSlot"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Slot
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="beaconBadBlobSlot"
            render={(props) => (
              <DebouncedInput<'beaconBadBlobSlot'>
                controllerProps={props}
                type="text"
                name="beaconBadBlobSlot"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      <div className="sm:col-span-2">
        <label
          htmlFor="beaconBadBlobEpoch"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Epoch
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="beaconBadBlobEpoch"
            render={(props) => (
              <DebouncedInput<'beaconBadBlobEpoch'>
                controllerProps={props}
                type="text"
                name="beaconBadBlobEpoch"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      <div className="sm:col-span-2">
        <label
          htmlFor="beaconBadBlobIndex"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Index
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="beaconBadBlobIndex"
            render={(props) => (
              <DebouncedInput<'beaconBadBlobIndex'>
                controllerProps={props}
                type="text"
                name="beaconBadBlobIndex"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      <div className="sm:col-span-4">
        <label
          htmlFor="beaconBadBlobBlockRoot"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Block root
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="beaconBadBlobBlockRoot"
            render={(props) => (
              <DebouncedInput<'beaconBadBlobBlockRoot'>
                controllerProps={props}
                type="text"
                name="beaconBadBlobBlockRoot"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      {errorComp && <div className="sm:col-span-6">{errorComp}</div>}
      <div className="sm:col-span-4">
        <CustomCombobox
          name="beaconBadBlobNode"
          disabled={isLoading || Boolean(error) || !data}
          label="Node"
          items={data?.node ?? []}
          itemToString={(item) => item}
        />
      </div>
      <div className="sm:col-span-2">
        <CustomCombobox
          name="beaconBadBlobNodeImplementation"
          disabled={isLoading || Boolean(error) || !data}
          label="Beacon Node Implementation"
          items={data?.beacon_implementation ?? []}
          itemToString={(item) => item}
        />
      </div>
      <div className="sm:col-span-4">
        <CustomCombobox
          name="beaconBadBlobNodeVersion"
          disabled={isLoading || Boolean(error) || !data}
          label="Beacon Node Version"
          items={data?.node_version ?? []}
          itemToString={(item) => item}
        />
      </div>
    </div>
  );
}
