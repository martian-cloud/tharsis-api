import { describe, expect, it } from 'vitest';
import { toSameOriginPath } from './returnPath';

describe('toSameOriginPath', () => {
    it.each([
        ['/groups/a/b', '/groups/a/b'],
        ['/groups/a/-/runs?tab=plan#logs', '/groups/a/-/runs?tab=plan#logs'],
        ['/', '/'],
    ])('keeps the in-app path %s', (path, expected) => {
        expect(toSameOriginPath(path)).toBe(expected);
    });

    it.each([
        '//evil.com/x',
        '/\\evil.com/x',
        'https://evil.com/x',
        'javascript:alert(1)',
    ])('falls back to / for the off-site path %s', (path) => {
        expect(toSameOriginPath(path)).toBe('/');
    });

    it('collapses a same-origin path that normalizes to a leading //', () => {
        expect(toSameOriginPath('/.//evil.com/x')).toBe('/evil.com/x');
    });

    it.each([null, undefined, ''])('falls back to / when there is no stored path (%s)', (path) => {
        expect(toSameOriginPath(path)).toBe('/');
    });
});
