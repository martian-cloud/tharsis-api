import Chip from '@mui/material/Chip';

interface Props {
    disabled: boolean
}

function RunnerChip({ disabled }: Props) {
    return (
        <Chip
            size="small"
            variant="outlined"
            color={disabled ? 'error' : 'default'}
            label={disabled ? 'Disabled' : 'Enabled'}
        />
    );
}

export default RunnerChip
