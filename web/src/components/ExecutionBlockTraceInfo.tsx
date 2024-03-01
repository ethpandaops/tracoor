export default function ExecutionBlockTraceInfo() {
  return (
    <div className="mx-2 mt-8 rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50">
      <h3 className="text-base font-semibold leading-6">
        Execution block traces are caputred from multiple sources via the{' '}
        <a
          href="https://geth.ethereum.org/docs/interacting-with-geth/rpc/ns-debug#debugtraceblock"
          className="text-amber-200 hover:text-amber-300 text-bold"
        >
          debug_traceBlock
        </a>{' '}
        RPC method as JSON objects.
      </h3>
    </div>
  );
}
