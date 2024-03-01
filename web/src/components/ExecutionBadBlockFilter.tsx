import { useFormContext, Controller } from 'react-hook-form';

import Alert from '@components/Alert';
import { CustomCombobox } from '@components/Combobox';
import DebouncedInput from '@components/DebouncedInput';
import { useUniqueExecutionBadBlockValues } from '@hooks/useQuery';

export default function FilterForm() {
  const { control } = useFormContext();

  const { data, isLoading, error } = useUniqueExecutionBadBlockValues([
    'node',
    'node_version',
    'execution_implementation',
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
    <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-6">
      <div className="sm:col-span-3 sm:col-start-1">
        <label
          htmlFor="executionBadBlockBlockHash"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Block hash
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="executionBadBlockBlockHash"
            render={(props) => (
              <DebouncedInput<'executionBadBlockBlockHash'>
                controllerProps={props}
                type="text"
                name="executionBadBlockBlockHash"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      <div className="sm:col-span-3">
        <label
          htmlFor="executionBadBlockBlockNumber"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Block number
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="executionBadBlockBlockNumber"
            render={(props) => (
              <DebouncedInput<'executionBadBlockBlockNumber'>
                controllerProps={props}
                type="text"
                name="executionBadBlockBlockNumber"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      {errorComp && <div className="sm:col-span-6">{errorComp}</div>}
      <div className="sm:col-span-2">
        <CustomCombobox
          name="executionBadBlockNode"
          disabled={isLoading || Boolean(error) || !data}
          label="Node"
          items={data?.node ?? []}
          itemToString={(item) => item}
        />
      </div>
      <div className="sm:col-span-2">
        <CustomCombobox
          name="executionBadBlockNodeImplementation"
          disabled={isLoading || Boolean(error) || !data}
          label="Execution Implementation"
          items={data?.execution_implementation ?? []}
          itemToString={(item) => item}
        />
      </div>
      <div className="sm:col-span-2">
        <CustomCombobox
          name="executionBadBlockNodeVersion"
          disabled={isLoading || Boolean(error) || !data}
          label="Execution Node Version"
          items={data?.node_version ?? []}
          itemToString={(item) => item}
        />
      </div>
    </div>
  );
}
