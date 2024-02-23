import { useState, useMemo } from 'react';

import SyntaxHighlighter from 'react-syntax-highlighter';
import { railscasts } from 'react-syntax-highlighter/dist/esm/styles/hljs';

import { ExecutionBlockTrace } from '@app/types/api';

export default function ExecutionBlockTraceEVMLabs({
  primaryTrace,
  relatedTraces,
}: {
  primaryTrace: ExecutionBlockTrace;
  relatedTraces: ExecutionBlockTrace[];
}) {
  const [step, setStep] = useState<number>(1);
  const [selected, setSelected] = useState<ExecutionBlockTrace[]>([primaryTrace]);
  const [tx, setTx] = useState<string>('');

  function generateOption(trace: ExecutionBlockTrace) {
    return (
      <div className="relative flex items-start pb-4 pt-3.5">
        <div className="min-w-0 flex-1 text-sm leading-6">
          <label htmlFor={trace.id} className="font-medium text-gray-900">
            {trace.node}
          </label>
          <p id={`${trace.id}-description`} className="text-gray-500">
            {trace.node_version}
          </p>
        </div>
        <div className="ml-3 flex h-6 items-center">
          <input
            id={trace.id}
            aria-describedby={`${trace.id}-description`}
            name="comments"
            type="checkbox"
            checked={selected.some((s) => s.id === trace.id)}
            disabled={selected.length >= 2 && !selected.some((s) => s.id === trace.id)}
            onChange={(e) => {
              if (e.target.checked) {
                setSelected((prev) => [...prev, trace]);
              } else {
                setSelected((prev) => prev.filter((s) => s.id !== trace.id));
              }
            }}
            className="h-4 w-4 rounded border-gray-300 text-sky-600 focus:ring-sky-600"
          />
        </div>
      </div>
    );
  }

  const step1 = (
    <div>
      <h3 className="text-base font-semibold leading-6 text-gray-900 py-2">
        Select two traces to compare
      </h3>
      <fieldset className="border-b border-t border-gray-200">
        <legend className="sr-only">Notifications</legend>
        <div className="divide-y divide-gray-200">
          {generateOption(primaryTrace)}
          {relatedTraces.map((trace) => generateOption(trace))}
        </div>
      </fieldset>
      <div className="flex justify-end m-2">
        <button
          type="button"
          disabled={selected.length !== 2}
          onClick={() => setStep(2)}
          className="rounded-md bg-sky-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-sky-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-600 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Next
        </button>
      </div>
    </div>
  );

  const step2 = (
    <div>
      <label htmlFor="tx" className="block text-base font-semibold leading-6 text-gray-900">
        Transaction index in the block (starting from 0)
      </label>
      <div className="mt-2">
        <input
          type="text"
          name="tx"
          id="tx"
          value={tx}
          onChange={(e) => setTx(e.target.value)}
          className="block w-full rounded-md border-0 pl-2 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-gray-600 sm:text-sm sm:leading-6"
        />
      </div>
      <div className="flex justify-end m-2">
        <button
          type="button"
          disabled={selected.length !== 2}
          onClick={() => setStep(1)}
          className="rounded-md mr-2 bg-amber-500 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-amber-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-amber-600 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Back
        </button>
        <button
          type="button"
          disabled={tx.length === 0}
          onClick={() => setStep(3)}
          className="rounded-md bg-sky-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-sky-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-600 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Next
        </button>
      </div>
    </div>
  );

  function generateJQCommand(trace: ExecutionBlockTrace, tx: string) {
    const nestedResult = !['nethermind', 'besu'].includes(trace.execution_implementation);
    return `jq '${nestedResult ? `{results: .[${tx}]}` : `.[${tx}]`}' ${trace.id}.json > ${trace.id}-${tx}.json`;
  }

  function generateFileNamePrefix(trace: ExecutionBlockTrace) {
    return `execution_block_trace-${trace.block_number}-${trace.block_hash}-${trace.node}`;
  }

  const snippet = useMemo(() => {
    if (selected.length !== 2) return '';
    if (tx.length === 0) return '';
    return `# Download the traces
# Note: requires wget
wget -O ${generateFileNamePrefix(selected[0])}.json.gz -q ${window.location.origin}/download/execution_block_trace/${selected[0].id}
wget -O ${generateFileNamePrefix(selected[1])}.json.gz -q ${window.location.origin}/download/execution_block_trace/${selected[1].id}

# Decompress the traces
gzip -d ${generateFileNamePrefix(selected[0])}.json.gz
gzip -d ${generateFileNamePrefix(selected[1])}.json.gz

# Pull out the transaction
# Note: requires jq
${generateJQCommand(selected[0], tx)}
${generateJQCommand(selected[1], tx)}

# Compare the traces
# Note: requires go and the tracediff tool
#       go install github.com/holiman/goevmlab/cmd/tracediff@latest
tracediff ${generateFileNamePrefix(selected[0])}-${tx}.json ${generateFileNamePrefix(selected[1])}-${tx}.json`;
  }, [selected, tx]);

  const step3 = (
    <div>
      <h3 className="text-base font-semibold leading-6 text-gray-900 py-2">
        Run the following commands to extract and compare the transaction traces
      </h3>
      <SyntaxHighlighter language="bash" style={railscasts} showLineNumbers wrapLines>
        {snippet}
      </SyntaxHighlighter>
      <div className="flex justify-end m-2">
        <button
          type="button"
          disabled={selected.length !== 2}
          onClick={() => setStep(2)}
          className="rounded-md bg-amber-500 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-amber-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-amber-600 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Back
        </button>
      </div>
    </div>
  );

  const currentStep = useMemo(() => {
    if (step <= 1) return step1;
    if (step === 2) return step2;
    return step3;
  }, [step, step1, step2, step3]);

  return <div className="px-10 mt-10">{currentStep}</div>;
}
