import { ReactNode } from 'react';

import FiltersProvider, { Props as FiltersProps } from '@providers/filters';
import NetworkProvider, { Props as NetworkProps } from '@providers/network';
import SelectionProvider, { Props as SelectionProps } from '@providers/selection';

interface Props {
  children: ReactNode;
  selection: Omit<SelectionProps, 'children'>;
  network?: Omit<NetworkProps, 'children'>;
  filters?: Omit<FiltersProps, 'children'>;
}

function Provider({ children, selection, filters, network }: Props) {
  return (
    <NetworkProvider {...network}>
      <SelectionProvider {...selection}>
        <FiltersProvider {...filters}>{children}</FiltersProvider>
      </SelectionProvider>
    </NetworkProvider>
  );
}

export default Provider;
