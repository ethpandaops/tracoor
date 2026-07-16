export default function ExecutionPayloadEnvelopeInfo() {
  return (
    <div className="mx-2 mt-8 rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50">
      <h3 className="text-base font-semibold leading-6">
        <a
          href="https://github.com/ethereum/consensus-specs/blob/master/specs/gloas/beacon-chain.md"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          Execution payload envelopes
        </a>{' '}
        are the signed gloas/ePBS objects that carry the execution payload contents, revealed by the
        builder separately from the beacon block, as bytes serialized by SSZ.
      </h3>
    </div>
  );
}
