export default function NCLIInfo() {
  return (
    <div className="mx-2 mt-8 rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50">
      <h3 className="text-base font-semibold leading-6">
        <a
          href="https://github.com/protolambda/zcli"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          zcli
        </a>{' '}
        is a Debugging command line tool, to work with SSZ files, process ETH 2.0 state transitions,
        and compute proofs and meta-data.
      </h3>

      <h3 className="text-base font-semibold leading-6 pt-5">
        You can use{' '}
        <a
          href="https://github.com/protolambda/zcli"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          zcli
        </a>{' '}
        to diff spec data, especially beacon states.
      </h3>
    </div>
  );
}
