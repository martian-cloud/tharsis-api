import React from 'react';
import { createRoot } from 'react-dom/client';
import graphQLFetcher from './api/fetchGraphQL';
import App from './App';
import { BasicAuthenticationService, OidcAuthenticationService } from './auth/AuthenticationService';
import config from './common/config';
import environment from './RelayEnvironment';
import reportWebVitals from './reportWebVitals';

const authSettingsQuery = `query srcQuery {
  authSettings {
    authType
    oidc {
        issuerUrl
        clientId
        scope
    }
  }
}`;

// oidcAuthType will be set to OIDC when an OIDC provider is configured
const oidcAuthType = 'OIDC';

const container = document.getElementById('root');

// Use createRoot to enable React concurrent mode
if (!container) throw new Error('Failed to find the root element');
const root = createRoot(container);

const graphqlEndpoint = `${config.apiUrl}/graphql`;

fetch(graphqlEndpoint, {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
    },
    credentials: 'omit',
    body: JSON.stringify({
        query: authSettingsQuery,
        variables: {},
    }),
}).then(async response => {
    const { authSettings } = (await response.json()).data;

    const authService = authSettings.authType === oidcAuthType ? new OidcAuthenticationService(
        authSettings.oidc.issuerUrl,
        authSettings.oidc.clientId,
        authSettings.oidc.scope
    ) : new BasicAuthenticationService();

    await authService.initialize();

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
