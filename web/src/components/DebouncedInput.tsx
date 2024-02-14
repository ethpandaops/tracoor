import React, { useState, useEffect, useCallback, SyntheticEvent, ChangeEvent } from 'react';

import {
  ControllerRenderProps,
  ControllerFieldState,
  UseFormStateReturn,
  FieldValues,
} from 'react-hook-form';
import { useDebouncedCallback } from 'use-debounce';

type InputProps = React.DetailedHTMLProps<
  React.InputHTMLAttributes<HTMLInputElement>,
  HTMLInputElement
>;

interface ControllerProps<TName extends string> {
  field: ControllerRenderProps<FieldValues, TName>;
  fieldState: ControllerFieldState;
  formState: UseFormStateReturn<FieldValues>;
}

type Props<TName extends string> = {
  wait?: number;
  controllerProps: ControllerProps<TName>;
} & InputProps;

function DebouncedTextField<TName extends string>({
  controllerProps,
  wait = 500,
  ...props
}: Props<TName>) {
  const { field, formState } = controllerProps;
  const [innerValue, setInnerValue] = useState('');

  useEffect(() => {
    if (field.value && typeof field.value === 'string') setInnerValue(field.value);
  }, [field.value, field.name]);

  const debouncedHandleChange = useDebouncedCallback((event) => {
    field.onChange(event);
    field.onBlur();
  }, wait);

  const handleChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      event.persist();
      setInnerValue(event.target.value);
      debouncedHandleChange(event);
    },
    [debouncedHandleChange],
  );

  return <input {...props} onChange={handleChange} value={innerValue} />;
}

export default DebouncedTextField;
