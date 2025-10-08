import { LoadingButton } from '@mui/lab';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Button from '@mui/material/Button';
import React from 'react';

interface Props {
    membership: any
    deleteInProgress: boolean;
    onClose: (confirm?: boolean) => void
}

function NamespaceMembershipDeleteConfirmationDialog(props: Props) {
    const { membership, deleteInProgress, onClose, ...other } = props;

    const member = membership.member.__typename === 'User' ? membership.member.username : membership.member.resourcePath;

    return (
        <Dialog
            maxWidth="xs"
            open={!!membership}
            keepMounted={false}
            {...other}
        >
            <DialogTitle>Remove Member</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to remove the member <strong>{member}</strong>?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton color="error" loading={deleteInProgress} onClick={() => onClose(true)}>Remove</LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default NamespaceMembershipDeleteConfirmationDialog;