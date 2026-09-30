import humanizeDuration from 'humanize-duration';

// humanize-duration has no shorthand option, so this humanizer registers a short English language
// ("1h 5m") for compact places like stat tiles and table cells.
const humanizer = humanizeDuration.humanizer({
    largest: 2,
    round: true,
    spacer: '',
    delimiter: ' ',
    units: ['h', 'm', 's', 'ms'],
    language: 'shortEn',
    languages: {
        shortEn: { h: () => 'h', m: () => 'm', s: () => 's', ms: () => 'ms' },
    },
});

// shortDuration renders a millisecond duration with shorthand units (e.g. "1h 5m", "450ms"), flooring
// sub-millisecond values at "<1ms"; pass options to override the defaults (e.g. { language: 'en' }).
export function shortDuration(ms: number, options?: humanizeDuration.Options): string {
    return ms > 0 && ms < 1 ? '<1ms' : humanizer(ms, options);
}
