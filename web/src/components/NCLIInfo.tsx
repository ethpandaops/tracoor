export default function NCLIInfo() {
  return (
    <div className="mx-2 mt-8 rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50">
      <h3 className="text-base font-semibold leading-6">
        <a
          href="https://github.com/status-im/nimbus-eth2/tree/stable/ncli"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          ncli
        </a>{' '}
        is a set of low level / debugging tools to interact with the nimbus{' '}
        <a
          href="https://github.com/ethereum/consensus-specs/tree/dev/specs"
          className="text-amber-200 hover:text-amber-300 text-bold"
        >
          beacon chain specification
        </a>{' '}
        implementation, similar to{' '}
        <a
          href="https://github.com/protolambda/zcli"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          zcli
        </a>
        . With it, you explore SSZ, make state transitions and compute hash tree roots.
      </h3>

      <h3 className="text-base font-semibold leading-6 pt-5">
        You can use{' '}
        <a
          href="https://github.com/status-im/nimbus-eth2/tree/stable/ncli"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          ncli
        </a>{' '}
        to perform state transition given a pre-state and a block to apply.
      </h3>
    </div>
  );
}
