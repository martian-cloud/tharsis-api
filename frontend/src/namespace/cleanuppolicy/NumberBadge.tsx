import { Box, Typography } from '@mui/material';

interface Props {
    value: number | string;
}

// NumberBadge renders a small circular badge with a number or short label inside, used for a rule's
// 1-based priority in the rules list and for the numbered steps in the rule dialog.
function NumberBadge({ value }: Props) {
    return (
        <Box sx={{
            width: 24, height: 24, borderRadius: '50%', bgcolor: 'action.selected',
            display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
        }}>
            <Typography variant="caption" fontWeight={600} lineHeight={1}>{value}</Typography>
        </Box>
    );
}

export default NumberBadge;
