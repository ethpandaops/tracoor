import { useContext as reactUseContext, createContext, useState } from 'react';

export const Context = createContext<State | undefined>(undefined);

export default function useContext() {
  const context = reactUseContext(Context);
  if (context === undefined) {
    throw new Error('Selection context must be used within a Selection provider');
  }
  return context;
}

export enum Selection {
  beacon_state = 'beacon_state',
  beacon_block = 'beacon_block',
  beacon_bad_block = 'beacon_bad_block',
  execution_block_trace = 'execution_block_trace',
  execution_bad_block = 'execution_bad_block',
  go_evm_lab_diff = 'go_evm_lab_diff',
  ncli_state_transition = 'ncli_state_transition',
  lcli_state_transition = 'lcli_state_transition',
  zcli_state_diff = 'zcli_state_diff',
}

export interface State {
  selection: Selection;
  setSelection: (selection: Selection) => void;
}

export interface ValueProps {
  selection: Selection;
}

export function useValue(props: ValueProps): State {
  const [selection, setSelection] = useState<Selection>(props.selection);

  return {
    selection,
    setSelection,
  };
}
