import { Box, FormControlLabel, Switch, Typography } from '@mui/material';

interface Props {
    locked: boolean;
    onChange: (event: any) => void;
}

function LockWorkspaceSetting(props: Props) {
    const { locked, onChange } = props;

    return (
        <Box sx={{ mb: 4 }}>
            <Typography variant="subtitle1" gutterBottom>Workspace Lock</Typography>
            <FormControlLabel
                control={<Switch sx={{ m: 2 }}
                    checked={locked}
                    color="secondary"
                    onChange={event => onChange(event)}
                />}
                label={locked ? "On" : "Off"}
            />
            <Typography variant="subtitle2">When enabled, this prevents new runs from starting and modifying the state version. A lock is often used to manually update the state version.</Typography>
        </Box>
    )
}

export default LockWorkspaceSetting;
