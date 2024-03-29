import '@styles/global.css';
import React from 'react';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import TimeAgo from 'javascript-time-ago';
import en from 'javascript-time-ago/locale/en.json';
import ReactDOM from 'react-dom/client';
import { Route, Switch } from 'wouter';

import App from '@app/App';
import ErrorBoundary from '@app/ErrorBoundary';
import { Selection } from '@contexts/selection';

const queryClient = new QueryClient();
TimeAgo.addDefaultLocale(en);

function isSelection(selection: string): selection is Selection {
  return Object.values(Selection).includes(selection as Selection);
}

if (process.env.NODE_ENV === 'development' && import.meta.env.VITE_MOCK) {
  const { worker } = await import('@app/mocks/browser');
  worker.start({
    waitUntilReady: true,
  });
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <Switch>
          <Route path="/:selection">
            {({ selection }) => {
              if (!isSelection(selection)) {
                return <App />;
              }
              return <App selection={selection} />;
            }}
          </Route>
          <Route path="/:selection/:id">
            {({ id, selection }) => {
              if (!isSelection(selection)) {
                return <App />;
              }
              return <App selection={selection} id={id} />;
            }}
          </Route>
          <Route>
            <App />
          </Route>
        </Switch>
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>,
);
