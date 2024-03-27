import { useFormContext, Controller } from 'react-hook-form';

import Alert from '@components/Alert';
import { CustomCombobox } from '@components/Combobox';
import DebouncedInput from '@components/DebouncedInput';
import { useUniqueExecutionBlockTraceValues } from '@hooks/useQuery';

export default function FilterForm() {
  const { control } = useFormContext();

  const { data, isLoading, error } = useUniqueExecutionBlockTraceValues([
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
          htmlFor="executionBlockTraceBlockHash"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Block hash
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="executionBlockTraceBlockHash"
            render={(props) => (
              <DebouncedInput<'executionBlockTraceBlockHash'>
                controllerProps={props}
                type="text"
                name="executionBlockTraceBlockHash"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      <div className="sm:col-span-3">
        <label
          htmlFor="executionBlockTraceBlockNumber"
          className="block text-sm font-bold leading-6 text-gray-700"
        >
          Block number
        </label>
        <div className="mt-2">
          <Controller
            control={control}
            name="executionBlockTraceBlockNumber"
            render={(props) => (
              <DebouncedInput<'executionBlockTraceBlockNumber'>
                controllerProps={props}
                type="text"
                name="executionBlockTraceBlockNumber"
                className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
              />
            )}
          />
        </div>
      </div>

      {errorComp && <div className="sm:col-span-6">{errorComp}</div>}
      <div className="sm:col-span-2">
        <CustomCombobox
          name="executionBlockTraceNode"
          disabled={isLoading || Boolean(error) || !data}
          label="Node"
          items={data?.node ?? []}
          itemToString={(item) => item}
        />
      </div>
      <div className="sm:col-span-2">
        <CustomCombobox
          name="executionBlockTraceNodeImplementation"
          disabled={isLoading || Boolean(error) || !data}
          label="Execution Implementation"
          items={data?.execution_implementation ?? []}
          itemToString={(item) => item}
        />
      </div>
      <div className="sm:col-span-2">
        <CustomCombobox
          name="executionBlockTraceNodeVersion"
          disabled={isLoading || Boolean(error) || !data}
          label="Execution Node Version"
          items={data?.node_version ?? []}
          itemToString={(item) => item}
        />
      </div>
    </div>
  );
}
