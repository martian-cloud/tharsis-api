import React from 'react';
import { createRoot } from 'react-dom/client';
import graphQLFetcher from './api/fetchGraphQL';
import App from './App';
import AuthenticationService from './auth/AuthenticationService';
import config from './common/config';
import environment from './RelayEnvironment';
import reportWebVitals from './reportWebVitals';

const authSettingsQuery = `query srcQuery {
  authSettings {
    oidcIssuerUrl
    oidcClientId
    oidcScope
  }
}`;

const container = document.getElementById('root');

// Use createRoot to enable React concurrent mode
if (!container) throw new Error('Failed to find the root element');
const root = createRoot(container);

const graphqlEndpoint = `${config.apiUrl}/graphql`;

fetch(graphqlEndpoint, {
    method: 'POST',
    credentials: 'omit',
    headers: {
        'Content-Type': 'application/json',
    },
    body: JSON.stringify({
        query: authSettingsQuery,
        variables: {},
    }),
}).then(async response => {
    const { authSettings } = (await response.json()).data;

    const authService = new AuthenticationService(
        authSettings.oidcIssuerUrl,
        authSettings.oidcClientId,
        authSettings.oidcScope
    );

    await authService.finishLogin();

    const fetchGraphQL = graphQLFetcher(authService);

    const relayEnv = environment(fetchGraphQL, authService);

    return root.render(
        <React.StrictMode>
            <App authService={authService} environment={relayEnv} />
        </React.StrictMode>
    );
}).catch(error => {
    const msg = `failed to query auth settings from ${graphqlEndpoint}: ${error}`;
    console.error(msg);
    root.render(
        <React.StrictMode>
            {msg}
        </React.StrictMode>
    );
});

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
