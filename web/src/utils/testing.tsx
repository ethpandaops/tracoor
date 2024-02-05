import { ReactNode } from 'react';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { Selection } from '@contexts/selection';
import ApplicationProvider from '@providers/application';
import { Props as SelectionProps } from '@providers/selection';

const queryClient = new QueryClient();

interface Props {
  selection: Omit<SelectionProps, 'children'>;
}

export function ProviderWrapper(
  { selection }: Props = { selection: { selection: Selection.beacon_state } },
) {
  return function Provider({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ApplicationProvider selection={selection}>{children}</ApplicationProvider>
      </QueryClientProvider>
    );
  };
}
