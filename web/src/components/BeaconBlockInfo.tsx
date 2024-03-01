export default function BeaconBlockInfo() {
  return (
    <div className="mx-2 mt-8 rounded-xl my-5 p-3 bg-sky-600 text-gray-100 font-bold border-4 border-gray-400/50">
      <h3 className="text-base font-semibold leading-6">
        <a
          href="https://eth2book.info/capella/part3/containers/blocks/"
          target="_blank"
          className="text-amber-100 hover:text-amber-200 text-bold bg-white/35 rounded-lg font-mono px-2 py-1"
          rel="noreferrer"
        >
          Beacon blocks
        </a>{' '}
        are captured from multiple sources via the{' '}
        <a
          href="https://ethereum.github.io/beacon-APIs/?urls.primaryName=dev#/Beacon/getBlockV2"
          className="text-amber-200 hover:text-amber-300 text-bold"
        >
          Beacon API beacon endpoint
        </a>{' '}
        as bytes serialized by SSZ.
      </h3>
    </div>
  );
}
