import { Box, Paper, Typography } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { ReactNode } from 'react';

// Card is the outlined container shared by the job detail sections.
export function Card({ children, sx }: { children: ReactNode; sx?: object }) {
    return (
        <Paper variant="outlined" sx={{ p: '20px 24px', borderRadius: 2, height: '100%', display: 'flex', flexDirection: 'column', ...sx }}>
            {children}
        </Paper>
    );
}

// CardTitle is a section heading with an optional right-aligned action/caption.
export function CardTitle({ children, action }: { children: ReactNode; action?: ReactNode }) {
    return (
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: '16px' }}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>{children}</Typography>
            {action}
        </Box>
    );
}

// InfoField is an icon + muted label stacked above a value, the row style used in the context cards.
export function InfoField({ icon, label, children, sx }: { icon: ReactNode; label: string; children: ReactNode; sx?: object }) {
    const theme = useTheme();

    return (
        <Box sx={{ mb: '18px', ...sx }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, color: theme.palette.text.secondary, mb: '4px' }}>
                <Box sx={{ display: 'flex' }}>{icon}</Box>
                <Typography variant="caption">{label}</Typography>
            </Box>
            <Box sx={{ color: theme.palette.text.primary }}>{children}</Box>
        </Box>
    );
}
