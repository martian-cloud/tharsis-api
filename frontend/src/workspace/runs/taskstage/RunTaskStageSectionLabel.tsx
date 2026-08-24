import { Typography, useTheme } from '@mui/material';
import { ReactNode } from 'react';

interface Props {
    children: ReactNode;
    // A resolved colour. Defaults to the disabled text colour, which is what a neutral heading uses.
    color?: string;
}

// RunTaskStageSectionLabel is the small uppercase heading above a block inside a policy panel or card.
function RunTaskStageSectionLabel({ children, color }: Props) {
    const theme = useTheme();
    return (
        <Typography
            variant="caption"
            component="div"
            sx={{
                fontWeight: 700,
                letterSpacing: '0.05em',
                textTransform: 'uppercase',
                color: color ?? theme.palette.text.disabled,
            }}
        >
            {children}
        </Typography>
    );
}

export default RunTaskStageSectionLabel;
