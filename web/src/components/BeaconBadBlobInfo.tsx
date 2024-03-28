export default function BeaconBadBlobInfo() {
  return (
    <div className="mx-2 mt-8 rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50">
      <h3 className="text-base font-semibold leading-6">
        Beacon bad blobs are{' '}
        <a
          href="https://github.com/ethereum/consensus-specs/blob/dev/specs/deneb/p2p-interface.md#blobsidecar"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          Beacon blobs
        </a>{' '}
        captured from beacon nodes where the blob succeeds gossip validation whilst failing full
        validation.
      </h3>
    </div>
  );
}
