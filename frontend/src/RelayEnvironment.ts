import { Environment, Network, Observable, RecordSource, RequestParameters, Store, Variables } from 'relay-runtime';
import { createClient } from 'graphql-ws';
import AuthenticationService from './auth/AuthenticationService';
import cfg from './common/config';

const KEEP_ALIVE_INTERVAL_MS = 60 * 1000; // send a ping to the server every minute
const PONG_TIMEOUT_MS = 10 * 1000; // close and reconnect if the server doesn't answer a ping
const RETRY_BASE_DELAY_MS = 1000; // first reconnect waits ~1s, then doubles each attempt
const MAX_RETRY_DELAY_MS = 30 * 1000; // cap the exponential backoff at 30s
const RETRY_JITTER_MS = 1000; // random jitter added to each delay to avoid thundering herd
// Keep the socket open for a while after the last subscription ends, so navigating
// away from a run and back reuses the connection instead of reconnecting.
const LAZY_CLOSE_TIMEOUT_MS = 60 * 1000;

const environment = (fetchGraphQL: (query: string, variables?: object) => Promise<any>, authService: AuthenticationService) => {
    let pongTimer: ReturnType<typeof setTimeout> | undefined;

    // The browser authenticates the WebSocket via the session cookie sent on the
    // upgrade request, so no token is passed in connectionParams.
    const subscriptionClient = createClient({
        url: `${cfg.wsUrl}/graphql`,
        lazy: true, // connect only when the first subscription is created
        lazyCloseTimeout: LAZY_CLOSE_TIMEOUT_MS, // debounce close so quick re-navigation reuses the socket
        keepAlive: KEEP_ALIVE_INTERVAL_MS,
        // Reconnect indefinitely while the tab is open; backoff caps at 30s. Fatal close
        // codes (4400/4401/4406/4409/4429) stop retries; 4403 is deliberately retried by
        // the client, since access may be granted on a later attempt.
        retryAttempts: Infinity,
        shouldRetry: () => true,
        // Refresh the session before each reconnect so the next upgrade carries a valid
        // cookie. A real expiry is handled in refreshSession (redirects on 401); other
        // errors are transient, so swallow and let the retry proceed.
        retryWait: async (retries) => {
            await authService.refreshSession().catch(() => { /* transient; retry handles it */ });
            // Exponential backoff with jitter.
            const delay = Math.min(RETRY_BASE_DELAY_MS * 2 ** retries, MAX_RETRY_DELAY_MS) + Math.random() * RETRY_JITTER_MS;
            await new Promise(resolve => setTimeout(resolve, delay));
        },
        on: {
            ping: (received) => {
                // keepAlive sends pings but the client doesn't time out on a missing pong
                // by itself, so start a timer; if no pong arrives, terminate and reconnect.
                if (!received) {
                    clearTimeout(pongTimer);
                    pongTimer = setTimeout(() => subscriptionClient.terminate(), PONG_TIMEOUT_MS);
                }
            },
            pong: (received) => {
                if (received) {
                    clearTimeout(pongTimer);
                }
            },
            closed: () => {
                clearTimeout(pongTimer);
            },
        },
    });

    const subscribe = (request: RequestParameters, variables: Variables) => {
        return Observable.create<any>(sink => {
            if (!request.text) {
                sink.error(new Error('Subscription is missing a query document'));
                return;
            }
            // subscribe returns a dispose function; Relay calls it on unsubscribe,
            // which makes the client send complete.
            return subscriptionClient.subscribe(
                {
                    operationName: request.name,
                    query: request.text,
                    variables,
                },
                {
                    next: sink.next.bind(sink),
                    error: sink.error.bind(sink),
                    complete: sink.complete.bind(sink),
                },
            );
        });
    };

    return new Environment({
        network: Network.create(
            async (params: any, variables) => {
                return fetchGraphQL(params.text, variables);
            },
            subscribe),
        store: new Store(new RecordSource()),
        UNSTABLE_defaultRenderPolicy: 'full',
    });
}

export default environment
