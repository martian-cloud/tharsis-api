import { Box, LinearProgress, Tooltip, Typography } from '@mui/material';
import { useTheme, Theme } from '@mui/material/styles';
import { ReactNode } from 'react';

// UsageBand maps a lower percent-of-limit bound to the bar color used at or above it. Bands are
// evaluated highest-first, so the first whose minPercent is met wins.
export interface UsageBand {
    minPercent: number;
    color: (theme: Theme) => string;
}

// DEFAULT_USAGE_BANDS escalates the bar through four colors as usage approaches the limit:
// green below 80%, amber at 80%, orange at 90%, red at 95%.
export const DEFAULT_USAGE_BANDS: UsageBand[] = [
    { minPercent: 95, color: (t) => t.palette.error.main },
    { minPercent: 90, color: (t) => t.palette.warning.main },
    { minPercent: 80, color: (t) => t.palette.warning.light },
    { minPercent: 0, color: (t) => t.palette.success.main },
];

function bandColor(theme: Theme, percent: number, bands: UsageBand[]): string {
    const band = bands.find((b) => percent >= b.minPercent) ?? bands[bands.length - 1];
    return band.color(theme);
}

export interface JobMetricTileProps {
    // Icon shown next to the label.
    icon: ReactNode;
    // Short label for the metric (e.g. "Peak memory").
    label: string;
    // Preformatted value shown when there is no limit to chart against.
    value: string;
    // Raw usage; with a positive limit the tile renders a usage-vs-limit bar instead of the value.
    usage?: number;
    // Raw limit; a null/zero limit falls back to the plain value.
    limit?: number | null;
    // Formats raw usage/limit numbers for the bar (defaults to locale-grouped integers).
    format?: (n: number) => string;
    // Color bands keyed by percent-of-limit; defaults to DEFAULT_USAGE_BANDS (green/amber/orange/red).
    bands?: UsageBand[];
}

function UsageBar({ usage, limit, format, bands }: {
    usage: number;
    limit: number;
    format: (n: number) => string;
    bands: UsageBand[];
}) {
    const theme = useTheme();

    const percent = limit > 0 ? (usage / limit) * 100 : 0;
    const clamped = Math.min(percent, 100);
    const color = bandColor(theme, percent, bands);

    const usageText = format(usage);
    const limitText = format(limit);

    return (
        <Box sx={{ minWidth: 0 }}>
            <Tooltip title={`${usageText} / ${limitText}`}>
                <Typography component="div" sx={{ fontWeight: 700, fontSize: '1.375rem', lineHeight: 1.1, mb: '12px', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {usageText}
                </Typography>
            </Tooltip>
            <LinearProgress
                variant="determinate"
                value={clamped}
                sx={{
                    height: 6,
                    borderRadius: 3,
                    backgroundColor: theme.palette.action.hover,
                    '& .MuiLinearProgress-bar': { backgroundColor: color },
                }}
            />
            <Typography variant="caption" sx={{ display: 'block', mt: '4px', textAlign: 'right', whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis', color: theme.palette.text.secondary }}>
                <Box component="span" sx={{ fontWeight: 600, color }}>{`${Math.round(percent)}%`}</Box>
                {` of ${limitText}`}
            </Typography>
        </Box>
    );
}

// JobMetricTile is a fixed-size stat tile. With a positive limit it charts usage against the limit as a
// colored bar; otherwise it shows the preformatted value as a plain stat (with a tooltip for the full,
// possibly-truncated, value).
function JobMetricTile({ icon, label, value, usage, limit, format = (n) => n.toLocaleString(), bands = DEFAULT_USAGE_BANDS }: JobMetricTileProps) {
    const theme = useTheme();
    const showBar = usage != null && limit != null && limit > 0;

    return (
        <Box sx={{
            height: '100%',
            minWidth: 0,
            minHeight: 92,
            p: '14px 16px',
            borderRadius: 2,
            border: `1px solid ${theme.palette.divider}`,
            boxSizing: 'border-box',
        }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, color: theme.palette.text.secondary, mb: '10px', minWidth: 0 }}>
                {icon}
                <Typography variant="caption" noWrap>{label}</Typography>
            </Box>
            {showBar ? (
                <UsageBar usage={usage} limit={limit} format={format} bands={bands} />
            ) : (
                <Tooltip title={value}>
                    <Typography
                        component="div"
                        sx={{ fontWeight: 700, fontSize: '1.375rem', lineHeight: 1.1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}
                    >
                        {value}
                    </Typography>
                </Tooltip>
            )}
        </Box>
    );
}

export default JobMetricTile;
