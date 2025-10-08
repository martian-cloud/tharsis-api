import React from 'react';
import { Box, Typography, Divider } from '@mui/material';

interface Props {
    title: string;
    children: React.ReactNode;
    isLast?: boolean;
}

function PreferenceSection({ title, children, isLast = false }: Props) {
    return (
        <Box sx={{ mb: isLast ? 0 : 4 }}>
            <Typography variant="h6" gutterBottom sx={{ mb: 2 }}>
                {title}
            </Typography>
            {children}
            {!isLast && <Divider sx={{ mt: 4 }} />}
        </Box>
    );
}

export default PreferenceSection;
