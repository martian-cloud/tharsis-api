import LoadingButton from '@mui/lab/LoadingButton';
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import { MutationError } from '../common/error';
import ServiceAccountAutocomplete, { ServiceAccountOption } from '../namespace/members/ServiceAccountAutocomplete';
import { useState } from 'react';

interface Props {
    error?: MutationError | null
    namespacePath: string
    assignCommitInFlight: boolean
    onClose: (selected?: ServiceAccountOption) => void
}

function AssignServiceAccountDialog({ onClose, assignCommitInFlight, error, namespacePath }: Props) {
    const [selected, setSelected] = useState<ServiceAccountOption | null>(null);

    return (
        <Dialog
            fullWidth
            maxWidth="sm"
            open>
            <DialogTitle>
                Assign Service Account
            </DialogTitle>
            <DialogContent dividers>
                {error && <Alert sx={{ mb: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>}
                <ServiceAccountAutocomplete namespacePath={namespacePath} onSelected={setSelected} />
            </DialogContent>
            <DialogActions>
                <Button
                    size="small"
                    variant="outlined"
                    onClick={() => onClose()}
                    color="inherit"
                >
                    Cancel</Button>
                <LoadingButton
                    disabled={!selected || !!error}
                    loading={assignCommitInFlight}
                    size="small"
                    variant="contained"
                    color="primary"
                    sx={{ ml: 2 }}
                    onClick={() => onClose(selected as ServiceAccountOption)}
                >
                    Assign
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default AssignServiceAccountDialog
