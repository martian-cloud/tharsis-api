import { OidcClient, SigninRequest } from "oidc-client-ts";
import { Cookies } from 'react-cookie';
import cfg from '../common/config';

const LOGIN_RETURN_TO = 'tharsis_oidc_login_return_to';
const CSRF_TOKEN_COOKIE_NAME = 'tharsis_csrf_token';
const CSRF_TOKEN_HEADER = 'X-Csrf-Token';

const cookies = new Cookies();

class AuthenticationService {
    private oidcClient: OidcClient;
    private pendingPromise: Promise<void> | null = null;

    constructor(issuerUrl: string, clientId: string, scope: string) {
        this.oidcClient = new OidcClient({
            authority: issuerUrl,
            client_id: clientId,
            scope: scope,
            redirect_uri: `${window.location.protocol}//${window.location.host}`
        });
    }

    async fetchWithAuth(input: string | URL | globalThis.Request, init: RequestInit | undefined): Promise<Response> {
        const csrfToken = cookies.get(CSRF_TOKEN_COOKIE_NAME);
        const fetchFn = () => fetch(input, { // nosemgrep: nodejs_scan.javascript-ssrf-rule-node_ssrf
            ...init,
            credentials: 'include', // Include cookies in the request
            headers: {
                [CSRF_TOKEN_HEADER]: csrfToken,
                ...init?.headers
            }
        });

        const response = await fetchFn();
        if (response.status === 401) {
            // Attempt to refresh the session, this will redirect to the login page if there is no valid refresh token
            await this.refreshSession()
            // Retry the original request after refreshing the session
            return fetchFn();
        }

        return response;
    }

    async logout() {
        try {
            const csrfToken = cookies.get(CSRF_TOKEN_COOKIE_NAME);
            const session = await fetch(`${cfg.apiUrl}/v1/sessions/logout`, {
                credentials: 'include',
                method: 'POST',
                headers: {
                    [CSRF_TOKEN_HEADER]: csrfToken
                }
            });

            if (!session.ok) {
                throw new Error(`Failed to logout session: ${session.statusText}`);
            }

            const req = await this.oidcClient.createSignoutRequest();

            window.location.assign(req.url);
        } catch (error) {
            console.error('Failed to logout session:', error);
        }
    }

    refreshSession(): Promise<void> {
        // If there's a pending promise, return it
        if (this.pendingPromise) {
            return this.pendingPromise;
        }

        // Create new promise and store it
        this.pendingPromise = new Promise((resolve, reject) => {
            const csrfToken = cookies.get(CSRF_TOKEN_COOKIE_NAME);
            fetch(`${cfg.apiUrl}/v1/sessions/refresh`, {
                method: 'POST',
                credentials: 'include',
                headers: {
                    [CSRF_TOKEN_HEADER]: csrfToken
                }
            })
                .then(response => {
                    if (response.status === 401) {
                        this.startLogin();
                        return;
                    }

                    if (!response.ok) {
                        throw new Error(`Failed to refresh session: ${response.statusText}`);
                    }

                    this.pendingPromise = null;
                    resolve();
                })
                .catch(error => {
                    // Clear the pending promise on error
                    this.pendingPromise = null;
                    reject(error);
                });
        });

        return this.pendingPromise;
    }

    async startLogin(): Promise<void> {
        try {
            window.sessionStorage.setItem(LOGIN_RETURN_TO, location.pathname + location.search);

            // Generate the signin request
            const request: SigninRequest = await this.oidcClient.createSigninRequest({
                state: { some: 'data' }, // Optional state to pass through
            });

            // Redirect the user to the OIDC provider's login page
            window.location.assign(request.url);
        } catch (error) {
            console.error('Failed to initiate oauth authorization code flow:', error);
        }
    }

    async finishLogin(): Promise<void> {
        try {
            if (!this._hasAuthRedirectParams()) {
                return
            }

            // Process the signin response
            const response = await this.oidcClient.processSigninResponse(window.location.href);

            const returnToPath = window.sessionStorage.getItem(LOGIN_RETURN_TO);

            window.history.replaceState(
                {},
                document.title,
                returnToPath ?? window.location.pathname
            );

            if (!response.id_token) {
                throw new Error('Failed to login user because ID token is undefined. Ensure that the openid scope is specified in the oauth settings for the identity provider.');
            }

            // Make post request to api to create session
            const session = await fetch(`${cfg.apiUrl}/v1/sessions`, {
                credentials: 'include',
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    token: response.id_token
                }),
            });

            if (!session.ok) {
                throw new Error(`Failed to create session: ${session.statusText}`);
            }

            // Verify that csrf token has been set
            const csrfToken = cookies.get(CSRF_TOKEN_COOKIE_NAME);
            if (!csrfToken) {
                throw new Error('missing csrf token cookie');
            }
        } catch (error) {
            console.error('Failed to complete oauth authorization code flow:', error);
            throw error;
        }
    }

    _hasAuthRedirectParams(): boolean {
        // response_mode: query
        let searchParams = new URLSearchParams(window.location.search);
        if ((searchParams.get("code") || searchParams.get("error")) &&
            searchParams.get("state")) {
            return true;
        }

        // response_mode: fragment
        searchParams = new URLSearchParams(window.location.hash.replace("#", "?"));
        if ((searchParams.get("code") || searchParams.get("error")) &&
            searchParams.get("state")) {
            return true;
        }

        return false;
    }
}

export default AuthenticationService;
