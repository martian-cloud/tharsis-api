// toSameOriginPath returns the path to send the user to after login, or '/' when the stored path
// would leave this site. The stored path comes from the URL the user was on, so it is
// attacker-influenced: '//evil.com/x' or '/\evil.com' are protocol-relative URLs that
// window.location.assign follows off-site. Resolving the path against this origin and keeping only
// the path, query and hash means the result can only ever navigate within the app.
export function toSameOriginPath(path: string | null | undefined): string {
    if (!path) {
        return '/';
    }

    let url: URL;
    try {
        url = new URL(path, window.location.origin);
    } catch {
        return '/';
    }

    if (url.origin !== window.location.origin) {
        return '/';
    }

    // A same-origin URL can still normalize to a pathname with a leading '//' (e.g. '/.//evil.com'),
    // which would again be read as protocol-relative, so collapse the leading slashes.
    return url.pathname.replace(/^\/+/, '/') + url.search + url.hash;
}
