import { alpha, Chip, useTheme } from '@mui/material';
import React from 'react';
import { getCheckStatusLabel } from './checks';

interface Props {
    status: string;
    withBackground?: boolean;
}

function CheckStatusChip({ status, withBackground = false }: Props) {
    const theme = useTheme();
    const colorMap = theme.palette.checkResult;
    const color = colorMap[status as keyof typeof colorMap] ?? colorMap.unknown;

    return (
        <Chip
            label={getCheckStatusLabel(status)}
            size="small"
            variant="outlined"
            sx={{
                color: color,
                borderColor: color,
                ...(withBackground && { backgroundColor: alpha(color, 0.1) }),
            }}
        />
    );
}

export default CheckStatusChip;
