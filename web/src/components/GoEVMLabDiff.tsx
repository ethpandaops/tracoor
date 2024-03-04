import { useMemo } from 'react';

import { XMarkIcon } from '@heroicons/react/24/outline';
import { useFormContext } from 'react-hook-form';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { railscasts } from 'react-syntax-highlighter/dist/esm/styles/hljs';

import { ExecutionBlockTrace } from '@app/types/api';
import Alert from '@components/Alert';
import CopyToClipboard from '@components/CopyToClipboard';
import ExecutionBlockTraceSelector from '@components/ExecutionBlockTraceSelector';
import GoEVMLabSetup from '@components/GoEVMLabSetup';
import Loading from '@components/Loading';
import useNetwork from '@contexts/network';
import { useExecutionBlockTraces } from '@hooks/useQuery';

export default function GoEVMLabDiff() {
  const { register, watch, setValue } = useFormContext();
  const { network } = useNetwork();

  const [executionBlockTraceSelectorId1, executionBlockTraceSelectorId2, goEvmLabDiffTx] = watch([
    'executionBlockTraceSelectorId1',
    'executionBlockTraceSelectorId2',
    'goEvmLabDiffTx',
  ]);

  const {
    data: trace1Data,
    isLoading: trace1IsLoading,
    error: trace1Error,
  } = useExecutionBlockTraces(
    {
      network: network ? network : undefined,
      id: executionBlockTraceSelectorId2,
      pagination: {
        limit: 1,
      },
    },
    Boolean(executionBlockTraceSelectorId2),
  );

  const {
    data: trace2Data,
    isLoading: trace2IsLoading,
    error: trace2Error,
  } = useExecutionBlockTraces(
    {
      network: network ? network : undefined,
      id: executionBlockTraceSelectorId1,
      pagination: {
        limit: 1,
      },
    },
    Boolean(executionBlockTraceSelectorId1),
  );

  const trace1 = trace1Data?.[0];
  const trace2 = trace2Data?.[0];
  let trace1FileName = '';
  let trace2FileName = '';
  if (trace1) trace1FileName = generateFileNamePrefix(trace1);
  if (trace2) trace2FileName = generateFileNamePrefix(trace2);

  function generateFileNamePrefix(trace: ExecutionBlockTrace) {
    return `execution_block_trace-${trace.block_number}-${trace.block_hash}-${trace.node}`;
  }

  function generateJQCommand(trace: ExecutionBlockTrace, fileName: string, tx: string) {
    const nestedResult = !['nethermind', 'besu'].includes(trace.execution_implementation);
    return `jq '${nestedResult ? `{results: .[${tx}]}` : `.[${tx}]`}' ${fileName}.json > ${fileName}-${tx}.json`;
  }

  const cmd = useMemo(() => {
    if (trace1 && trace2 && goEvmLabDiffTx !== '' && goEvmLabDiffTx !== undefined) {
      return `# Download the state and block
# Note: requires wget
wget -O ${trace1FileName}.json.gz -q ${window.location.origin}/download/execution_block_trace/${trace1.id}
wget -O ${trace2FileName}.json.gz -q ${window.location.origin}/download/execution_block_trace/${trace2.id}

# Decompress the state and block
gzip -f -d ${trace1FileName}.json.gz
gzip -f -d ${trace2FileName}.json.gz

# Pull out the transaction
# Note: requires jq
${generateJQCommand(trace1, trace1FileName, goEvmLabDiffTx)}
${generateJQCommand(trace2, trace2FileName, goEvmLabDiffTx)}

# Compare the traces
# Note: requires go and the tracediff tool
#       go install github.com/holiman/goevmlab/cmd/tracediff@latest
tracediff ${trace1FileName}-${goEvmLabDiffTx}.json ${trace2FileName}-${goEvmLabDiffTx}.json`;
    }
    return '';
  }, [trace1Data, trace2Data, goEvmLabDiffTx]);

  let otherComp = undefined;

  if (trace1IsLoading || trace2IsLoading) {
    otherComp = <Loading />;
  } else if (trace1Error || trace2Error) {
    let message = 'Something went wrong fetching data';
    if (typeof trace1Error === 'string') {
      message = trace1Error;
    }
    if (typeof trace2Error === 'string') {
      message = trace2Error;
    }
    otherComp = <Alert type="error" message={message} />;
  } else if (cmd && !trace1) {
    otherComp = <Alert type="error" message="Execution block trace #1 data not found" />;
  } else if (cmd && !trace2) {
    otherComp = <Alert type="error" message="Execution block trace #2 data not found" />;
  }

  return (
    <div className="mx-2 mt-8">
      <GoEVMLabSetup />
      <ExecutionBlockTraceSelector num={1} excludeNum={2} />
      <ExecutionBlockTraceSelector num={2} excludeNum={1} />
      <div className="bg-white/35 my-10 px-8 py-5 rounded-xl">
        <div className="absolute -mt-8 bg-white/65 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold border-2 border-sky-400">
          Transaction index
        </div>
        {goEvmLabDiffTx && (
          <button
            className="absolute right-8 sm:right-14 -mt-8 bg-white/85 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-gray-600 font-bold flex cursor-pointer transition hover:text-gray-800 border-2 border-gray-500 hover:border-gray-700"
            onClick={() => setValue(`goEvmLabDiffTx`, '')}
          >
            Clear
            <XMarkIcon className="w-4 h-4" />
          </button>
        )}
        <h3 className="text-lg font-bold my-5 text-gray-700">
          Select the transaction index, starting from 0, to compare in each block trace
        </h3>
        <div className="bg-white/35 border-lg rounded-lg p-4">
          <label
            htmlFor="goEvmLabDiffTx"
            className="block text-sm font-bold leading-6 text-gray-700"
          >
            Transaction index
          </label>
          <div className="mt-2">
            <input
              {...register('goEvmLabDiffTx')}
              type="number"
              className="block w-full rounded-md border-0 bg-white/45 px-2.5 py-1.5 text-gray-600 shadow-sm ring-1 ring-inset ring-white/10 focus:ring-2 focus:ring-inset focus:ring-sky-500 sm:text-sm sm:leading-6"
            />
          </div>
        </div>
      </div>
      {(otherComp || cmd) && (
        <>
          <div className="bg-white/35 my-10 px-8 py-5 rounded-xl">
            <div className="absolute -mt-8 bg-white/65 px-3 py-1 -ml-6 shadow-xl text-xs rounded-lg text-sky-600 font-bold border-2 border-sky-400">
              Transaction tracediff Command
            </div>
            <div className="mt-2">
              {otherComp}
              {!otherComp && (
                <>
                  <div className="absolute right-14 sm:right-20 m-2 bg-white/35 mix-blend-hard-light hover:bg-white/20 rounded-lg cursor-pointer">
                    <CopyToClipboard text={cmd} className="m-2" inverted />
                  </div>
                  <SyntaxHighlighter language="bash" style={railscasts} showLineNumbers wrapLines>
                    {cmd}
                  </SyntaxHighlighter>
                </>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
