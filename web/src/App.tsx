import { Selection } from '@contexts/selection';
import Filters from '@parts/Filters';
import Header from '@parts/Header';
import Listing from '@parts/Listing';
import Selector from '@parts/Selector';
import ApplicationProvider from '@providers/application';

export default function App({
  selection = Selection.beacon_state,
  id,
}: {
  selection?: Selection;
  id?: string;
}) {
  return (
    <ApplicationProvider selection={{ selection }}>
      <div className="flex flex-col min-h-screen">
        <Header />
        <main className="flex-grow bg-gradient-to-r from-amber-400 to-amber-600">
          <Selector />
          <Filters />
          <Listing id={id} />
        </main>
      </div>
    </ApplicationProvider>
  );
}
