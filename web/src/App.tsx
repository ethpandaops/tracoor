import { Selection } from '@contexts/selection';
import Filters from '@parts/Filters';
import Header from '@parts/Header';
import Listing from '@parts/Listing';
import Selector from '@parts/Selector';
import ApplicationProvider from '@providers/application';

export default function App() {
  return (
    <ApplicationProvider selection={{ selection: Selection.beacon_state }}>
      <div className="h-screen w-screen flex flex-col">
        <Header />
        <main className="flex-grow bg-gradient-to-r from-amber-400 to-amber-600 h-full">
          <Selector />
          <Filters />
          <Listing />
        </main>
      </div>
    </ApplicationProvider>
  );
}
