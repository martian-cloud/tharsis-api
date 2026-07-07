import { Divider, Paper, Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import humanizeDuration from 'humanize-duration';
import React from 'react';

interface Props {
    icon: React.ReactNode;
    title: string;
    durationMs?: number | null;
    description?: string;
    actions?: React.ReactNode;
    children?: React.ReactNode;
}

// RunDetailsStageStatusCard is the always-visible status card on the plan/apply stage
// pages. The page skeleton stays fixed across run states: only this card's title,
// description, actions, and expansion content (error summary, plan summary) vary.
function RunDetailsStageStatusCard({ icon, title, durationMs, description, actions, children }: Props) {
    const theme = useTheme();

    return (
        <Paper variant="outlined" sx={{ marginBottom: 2, p: 2 }}>
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                alignItems: 'center',
                [theme.breakpoints.down('md')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *:not(:last-child)': {
                        marginBottom: 2
                    },
                }
            }}>
                <Box display="flex" alignItems="center">
                    {icon}
                    <Box>
                        <Typography variant="h6">{title}</Typography>
                        {!!durationMs && <Typography variant="body2" color="textSecondary">
                            Duration: {humanizeDuration(durationMs)}
                        </Typography>}
                        {description && <Typography variant="body2" color="textSecondary">
                            {description}
                        </Typography>}
                    </Box>
                </Box>
                {actions && <Box display="flex" gap={1}>
                    {actions}
                </Box>}
            </Box>
            {children && <React.Fragment>
                <Divider sx={{ ml: -2, mr: -2, mt: 2 }} />
                {children}
            </React.Fragment>}
        </Paper>
    );
}

export default RunDetailsStageStatusCard;
