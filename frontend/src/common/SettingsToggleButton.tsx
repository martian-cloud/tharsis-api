import { Box, Button, Typography } from '@mui/material';

interface Props {
    title: string;
    showSettings: boolean;
    onToggle: () => void;
}

function SettingsToggleButton({ title, showSettings, onToggle }: Props) {
    return (
        <Box sx={{ display: "flex", justifyContent: "space-between" }}>
            <Typography variant="h6" gutterBottom>{title}</Typography>
            <Box>
                <Button
                    color="info"
                    variant="outlined"
                    onClick={onToggle}
                >
                    {showSettings ? 'Hide' : 'Show'}
                </Button>
            </Box>
        </Box>
    );
}

export default SettingsToggleButton;
