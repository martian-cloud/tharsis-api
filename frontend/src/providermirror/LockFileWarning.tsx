import { Alert, AlertTitle, Box } from '@mui/material';

function LockFileWarning() {
    return (
        <Alert severity="info" variant="outlined" sx={{ mt: 2 }}>
            <AlertTitle>Expected Behavior</AlertTitle>
            Run logs may show <code style={{ color: 'inherit' }}><Box component="span" sx={{ color: 'warning.main' }}>Warning</Box>: Incomplete lock file information</code> — network mirrors only record checksums for the current platform. This is safe to ignore.
        </Alert>
    );
}

export default LockFileWarning;
