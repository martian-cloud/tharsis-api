// Shared layout components for run summary rows (plan changes, check results).
import MuiInfoIcon from '@mui/icons-material/Info';
import { Paper, Tooltip, useTheme } from '@mui/material';
import { darken, SxProps, Theme } from '@mui/material/styles';
import React from 'react';

export function InfoIcon({ text }: { text: string }) {
    return (
        <Tooltip title={text}>
            <MuiInfoIcon sx={{
                width: 16,
                height: 16,
                marginLeft: '10px',
                verticalAlign: 'middle',
                opacity: '20%',
                transition: 'ease',
                transitionDuration: '300ms',
                ":hover": {
                    opacity: '100%'
                }
            }} />
        </Tooltip>
    );
}

export function SummaryRowCol({ children, sx }: { children: React.ReactNode, sx?: SxProps<Theme> }) {
    const theme = useTheme();
    return (
        <Paper
            sx={{
                ...sx,
                minHeight: 41,
                pt: 1,
                pb: 1,
                backgroundColor: darken(theme.palette.background.default, 0.3),
                display: 'flex',
                alignItems: 'center',
                width: '100%',
                boxShadow: 'none',
                borderRadius: 0,
                borderBottom: `1px solid ${theme.palette.divider}`,
                '&:last-child': { borderBottom: 'none' }
            }}>
            {children}
        </Paper>
    );
}
