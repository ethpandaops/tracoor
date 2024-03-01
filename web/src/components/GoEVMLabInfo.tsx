export default function NCLIInfo() {
  return (
    <div className="mx-2 mt-8 rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50">
      <h3 className="text-base font-semibold leading-6">
        <a
          href="https://github.com/holiman/goevmlab"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          Go EVM Lab
        </a>{' '}
        is a minimal &quot;compiler&quot;, along with some tooling to view traces in a UI, and
        execute scripts against EVMs.
      </h3>

      <h3 className="text-base font-semibold leading-6 pt-5">
        Tracediff allows you to load evm files and find differences between transactions.
      </h3>
    </div>
  );
}
