import { ReactNode } from 'react';
import { Box, Stack, Typography } from '@mui/material';
import NumberBadge from './NumberBadge';

interface Props {
    step: number;
    title: string;
    // Rendered on the right of the header row, e.g. the Visual/JSON tab switch for the Rules step.
    action?: ReactNode;
    children: ReactNode;
}

// One numbered step (Type / Settings / Rules) of the New/Edit cleanup policy forms. Reuses the
// same NumberBadge + title pattern as CleanupRuleDialog's steps, so the whole feature — from the
// policy-level form down to an individual rule dialog — reads as one consistent design language.
function CleanupPolicyFormSection({ step, title, action, children }: Props) {
    return (
        <Box sx={{ mb: 4 }}>
            <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={1} sx={{ mb: 1.5 }}>
                <Stack direction="row" alignItems="center" spacing={1}>
                    <NumberBadge value={step} />
                    <Typography variant="subtitle2">{title}</Typography>
                </Stack>
                {action}
            </Stack>
            {children}
        </Box>
    );
}

export default CleanupPolicyFormSection;
