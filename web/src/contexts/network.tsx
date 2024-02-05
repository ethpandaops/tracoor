import { useContext as reactUseContext, createContext, useState } from 'react';

export const Context = createContext<State | undefined>(undefined);

export default function useContext() {
  const context = reactUseContext(Context);
  if (context === undefined) {
    throw new Error('Network context must be used within a Network provider');
  }
  return context;
}

export interface State {
  network?: string;
  setNetwork: (network: string) => void;
}

export interface ValueProps {
  network?: string;
}

export function useValue(props: ValueProps): State {
  const [network, setNetwork] = useState<string | undefined>(props.network);

  return {
    network,
    setNetwork,
  };
}
