import LoadingButton from "@mui/lab/LoadingButton";
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import React from "react";

interface Props {
    title: string;
    message: React.ReactNode;
    confirmButtonLabel: string;
    opInProgress: boolean;
    onConfirm: () => void;
    onClose: () => void;
}

function ConfirmationDialog({ title, message, confirmButtonLabel, opInProgress, onClose, onConfirm, ...other }: Props) {
    return (
        <Dialog
            maxWidth="xs"
            open
            keepMounted={false}
            {...other}
        >
            <DialogTitle>{title}</DialogTitle>
            <DialogContent dividers>
                {message}
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={onClose}>
                    No
                </Button>
                <LoadingButton loading={opInProgress} onClick={onConfirm}>
                    {confirmButtonLabel}
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default ConfirmationDialog;
