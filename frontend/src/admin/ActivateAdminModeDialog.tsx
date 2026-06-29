import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, FormControl, InputLabel, MenuItem, Select } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import { ActivateAdminModeDialogMutation } from './__generated__/ActivateAdminModeDialogMutation.graphql';

interface Props {
    open: boolean;
    onClose: () => void;
}

function ActivateAdminModeDialog({ open, onClose }: Props) {
    const [duration, setDuration] = useState(30);
    const [error, setError] = useState<string | null>(null);
    const { enqueueSnackbar } = useSnackbar();

    const [commit, isInFlight] = useMutation<ActivateAdminModeDialogMutation>(graphql`
        mutation ActivateAdminModeDialogMutation($input: ActivateAdminModeInput!) {
            activateAdminMode(input: $input) {
                user {
                    id
                    adminModeEnabled
                    adminModeExpiration
                }
                problems {
                    message
                    type
                    field
                }
            }
        }
    `);

    function handleActivate() {
        setError(null);
        commit({
            variables: { input: { durationMinutes: duration } },
            updater: (store) => {
                store.invalidateStore();
            },
            onCompleted: (data) => {
                if (data.activateAdminMode.problems.length === 0) {
                    onClose();
                    enqueueSnackbar('Admin mode activated', { variant: 'success' });
                } else {
                    setError(data.activateAdminMode.problems.map(p => p.message).join('; '));
                }
            },
        });
    }

    return (
        <Dialog open={open} onClose={onClose} maxWidth="xs" fullWidth>
            <DialogTitle>Activate Admin Mode</DialogTitle>
            <DialogContent dividers>
                {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
                <FormControl fullWidth sx={{ mt: 1 }}>
                    <InputLabel>Duration</InputLabel>
                    <Select
                        value={duration}
                        label="Duration"
                        onChange={(e) => setDuration(e.target.value as number)}
                    >
                        <MenuItem value={15}>15 minutes</MenuItem>
                        <MenuItem value={30}>30 minutes</MenuItem>
                        <MenuItem value={60}>1 hour</MenuItem>
                        <MenuItem value={120}>2 hours</MenuItem>
                        <MenuItem value={360}>6 hours</MenuItem>
                    </Select>
                </FormControl>
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={onClose}>Cancel</Button>
                <Button onClick={handleActivate} loading={isInFlight}>
                    Activate
                </Button>
            </DialogActions>
        </Dialog>
    );
}

export default ActivateAdminModeDialog;
