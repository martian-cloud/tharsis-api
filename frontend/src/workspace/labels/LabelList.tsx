import { useMemo } from 'react';
import { Box, Chip, Tooltip, Typography } from '@mui/material';
import LabelChip from './LabelChip';
import { Label } from './types';

interface Props {
    labels: Label[];
    maxVisible?: number;
    size?: 'small' | 'medium' | 'xs';
    variant?: 'filled' | 'outlined';
    spacing?: number;
    showEmptyState?: boolean;
    emptyStateText?: string;
    prefix?: string;
}

function LabelList({
    labels,
    maxVisible = 5,
    size = 'small',
    variant = 'outlined',
    spacing = 1,
    showEmptyState = false,
    emptyStateText = 'No labels',
    prefix
}: Props) {
    // Sort labels alphabetically by key to ensure consistent ordering
    // (Go maps are unordered, so we need to sort in the frontend)
    const sortedLabels = useMemo(() =>
        [...labels].sort((a, b) => a.key.localeCompare(b.key)),
        [labels]
    );

    // Calculate display mode
    const { showKeysOnly, visibleCount } = useMemo(() => {
        if (labels.length === 0) {
            return { showKeysOnly: false, visibleCount: 0 };
        }

        // Check if we need to show keys only based on total length
        const totalLength = sortedLabels.reduce((sum, label) =>
            sum + label.key.length + label.value.length, 0
        );
        const keysOnly = totalLength > 100;

        // Use maxVisible as the limit - let the layout handle wrapping
        const count = Math.min(sortedLabels.length, maxVisible);

        return { showKeysOnly: keysOnly, visibleCount: count };
    }, [labels.length, sortedLabels, maxVisible]);

    const visibleLabels = sortedLabels.slice(0, visibleCount);
    const hiddenCount = sortedLabels.length - visibleCount;

    // Handle empty state (after all hooks are called)
    if (labels.length === 0) {
        if (showEmptyState) {
            return (
                <Typography
                    variant="body2"
                    color="textSecondary"
                    sx={{ fontStyle: 'italic' }}
                >
                    {emptyStateText}
                </Typography>
            );
        }
        return null;
    }

    return (
        <Box
            sx={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: spacing,
                alignItems: 'center'
            }}
        >
            {prefix && (
                <Typography variant="body2" color="textSecondary" component="span">
                    {prefix}:
                </Typography>
            )}
            {visibleLabels.map((label, index) => (
                <LabelChip
                    key={`${label.key}-${index}`}
                    labelKey={label.key}
                    value={label.value}
                    size={size}
                    variant={variant}
                    showKeyOnly={showKeysOnly}
                />
            ))}
            {hiddenCount > 0 && (
                <Tooltip
                    title={
                        <Box>
                            {sortedLabels.slice(visibleCount).map((label, index) => (
                                <div key={index}>{label.key}: {label.value}</div>
                            ))}
                        </Box>
                    }
                    arrow
                >
                    <Chip
                        label={`+${hiddenCount} more`}
                        size={size}
                        variant={variant}
                        sx={{
                            fontSize: size === 'xs' ? '0.75rem' : undefined,
                            height: size === 'xs' ? '20px' : undefined,
                        }}
                    />
                </Tooltip>
            )}
            {showKeysOnly && (
                <Typography variant="caption" color="textSecondary" sx={{ fontStyle: 'italic' }}>
                    (hover for values)
                </Typography>
            )}
        </Box>
    );
}

export default LabelList;
