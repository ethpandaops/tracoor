import { useState } from 'react';

import { Combobox, Label } from '@headlessui/react';
import { CheckIcon, ChevronUpDownIcon } from '@heroicons/react/20/solid';
import classNames from 'classnames';
import { Controller, useFormContext } from 'react-hook-form';
interface CustomComboboxProps {
  label: string;
  name: string; // Add a name prop for react-hook-form
  items: string[];
  itemToString: (item: string) => string;
  disabled?: boolean;
}

export function CustomCombobox({
  label,
  name,
  items,
  itemToString,
  disabled = false,
}: CustomComboboxProps) {
  const { control } = useFormContext(); // Access the form control object
  const [query, setQuery] = useState('');

  return (
    <Controller
      name={name}
      control={control}
      render={({ field }) => (
        <Combobox
          as="div"
          value={field.value}
          onChange={(value) => {
            field.onChange(value); // Update react-hook-form's value when the combobox value changes
          }}
          nullable
          disabled={disabled}
        >
          <Label className="block text-sm font-bold leading-6 text-gray-600 truncate">
            {label}
          </Label>
          <div className="relative mt-2">
            <Combobox.Input
              className="w-full rounded-md border-0 bg-white/45 py-1.5 pl-3 pr-10 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6"
              onChange={(event) => {
                setQuery(event.target.value);
                // Optionally update the field value or perform additional actions here
              }}
              displayValue={(item: string) => itemToString(item) ?? ''}
            />
            <Combobox.Button className="absolute inset-y-0 right-0 flex items-center rounded-r-md px-2 focus:outline-none">
              <ChevronUpDownIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
            </Combobox.Button>

            {/* Filter items based on query */}
            <Combobox.Options className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-white py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none sm:text-sm">
              {items
                .filter((item) => itemToString(item).toLowerCase().includes(query.toLowerCase()))
                .map((item) => (
                  <Combobox.Option
                    key={item}
                    value={item}
                    className={({ focus }) =>
                      classNames(
                        'relative cursor-default select-none py-2 pl-8 pr-4',
                        focus ? 'bg-indigo-600 text-white' : 'text-gray-900',
                      )
                    }
                  >
                    {({ selected, focus }) => (
                      <>
                        <span className={classNames('block truncate', selected && 'font-semibold')}>
                          {itemToString(item)}
                        </span>
                        {selected && (
                          <span
                            className={classNames(
                              'absolute inset-y-0 left-0 flex items-center pl-1.5',
                              focus ? 'text-white' : 'text-indigo-600',
                            )}
                          >
                            <CheckIcon className="h-5 w-5" aria-hidden="true" />
                          </span>
                        )}
                      </>
                    )}
                  </Combobox.Option>
                ))}
            </Combobox.Options>
          </div>
        </Combobox>
      )}
    />
  );
}
