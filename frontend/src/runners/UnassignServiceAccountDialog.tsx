import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import LoadingButton from '@mui/lab/LoadingButton';
import { MutationError } from "../common/error";

interface Props {
    onClose: (confirm?: boolean) => void
    unAssignCommitInFlight: boolean
    error?: MutationError | null
    name: string
}

function UnassignServiceAccountDialog(props: Props) {
    const { onClose, unAssignCommitInFlight, error, name, ...other } = props;

    return (
        <Dialog
            maxWidth="sm"
            open
            keepMounted={false}
            {...other}
        >
            <DialogTitle>Unassign Service Account</DialogTitle>
            <DialogContent dividers>
            {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>}
                Are you sure you want to unassign service account <strong>{name}</strong> from this agent?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton
                    color="error"
                    loading={unAssignCommitInFlight}
                    onClick={() => onClose(true)}>
                    Unassign
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default UnassignServiceAccountDialog
