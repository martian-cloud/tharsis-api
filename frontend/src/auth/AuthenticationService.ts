import { OidcClient, SigninRequest } from "oidc-client-ts";
import { Cookies } from 'react-cookie';
import cfg from '../common/config';

const LOGIN_RETURN_TO = 'tharsis_oidc_login_return_to';
const CSRF_TOKEN_COOKIE_NAME = 'tharsis_csrf_token';
const CSRF_TOKEN_HEADER = 'X-Csrf-Token';

const cookies = new Cookies();

interface CreateSessionOptions {
    token?: string;
    username?: string;
    password?: string;
}

abstract class AuthenticationService {
    private pendingPromise: Promise<void> | null = null;

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

            await this._logout()
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
                        this.redirectToLogin();
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

    async redirectToLogin(): Promise<void> {
        try {
            window.sessionStorage.setItem(LOGIN_RETURN_TO, location.pathname + location.search);

            await this._redirectToLogin()
        } catch (error) {
            console.error('Failed to initiate oauth authorization code flow:', error);
        }
    }

    async login({ token, username, password }: CreateSessionOptions): Promise<void> {
        const session = await fetch(`${cfg.apiUrl}/v1/sessions`, {
            credentials: 'include',
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                token,
                username,
                password
            }),
        });

        if (!session.ok) {
            // Read response body for more details
            const errorMessage = await session.text();
            throw new Error(`Failed to create session: ${session.statusText}: ${errorMessage}`);
        }

        // Verify that csrf token has been set
        const csrfToken = cookies.get(CSRF_TOKEN_COOKIE_NAME);
        if (!csrfToken) {
            throw new Error('missing csrf token cookie');
        }

        const returnToPath = window.sessionStorage.getItem(LOGIN_RETURN_TO);

        if (returnToPath) {
            window.sessionStorage.removeItem(LOGIN_RETURN_TO);
        }

        this._redirectAfterLogin(returnToPath ?? '/')
    }

    initialize(): Promise<void> {
        // noop
        return Promise.resolve();
    }

    abstract _redirectToLogin(): Promise<void>;
    abstract _redirectAfterLogin(path: string): void;
    abstract _logout(): Promise<void>;
}

export class OidcAuthenticationService extends AuthenticationService {
    private oidcClient: OidcClient;

    constructor(issuerUrl: string, clientId: string, scope: string) {
        super();

        this.oidcClient = new OidcClient({
            authority: issuerUrl,
            client_id: clientId,
            scope: scope,
            redirect_uri: `${window.location.protocol}//${window.location.host}`
        });
    }

    async initialize(): Promise<void> {
        try {
            if (!this._hasAuthRedirectParams()) {
                return
            }

            // Process the signin response
            const response = await this.oidcClient.processSigninResponse(window.location.href);

            if (!response.id_token) {
                throw new Error('Failed to login user because ID token is undefined. Ensure that the openid scope is specified in the oauth settings for the identity provider.');
            }

            await this.login({ token: response.id_token });
        } catch (error) {
            console.error('Failed to complete oauth authorization code flow:', error);
            throw error;
        }
    }

    async _redirectToLogin(): Promise<void> {
        // Generate the signin request
        const request: SigninRequest = await this.oidcClient.createSigninRequest({
            state: { some: 'data' }, // Optional state to pass through
        });

        // Redirect the user to the OIDC provider's login page
        window.location.assign(request.url);
    }

    _redirectAfterLogin(path: string) {
        window.history.replaceState(
            {},
            document.title,
            path
        );
    }

    async _logout(): Promise<void> {
        const req = await this.oidcClient.createSignoutRequest();
        window.location.assign(req.url);
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

export class BasicAuthenticationService extends AuthenticationService {

    _redirectAfterLogin(path: string) {
        window.location.assign(path);
    }

    _redirectToLogin(): Promise<void> {
        window.location.assign('/login');
        return Promise.resolve()
    }

    _logout(): Promise<void> {
        window.location.assign('/login');
        return Promise.resolve();
    }
}

export default AuthenticationService;
