import LoadingButton from "@mui/lab/LoadingButton";
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import React from "react";

interface Props {
    title: string;
    children: React.ReactNode;
    confirmLabel: string;
    confirmDisabled?: boolean;
    confirmColor?: 'error' | 'primary';
    confirmInProgress: boolean;
    onConfirm: () => void;
    onClose: () => void;
    maxWidth?: 'xs' | 'sm' | 'md';
}

function ConfirmationDialog({ title, children, confirmLabel, confirmDisabled, confirmColor = 'error', confirmInProgress, onClose, onConfirm, maxWidth = 'xs', ...other }: Props) {
    return (
        <Dialog
            maxWidth={maxWidth}
            open
            keepMounted={false}
            {...other}
        >
            <DialogTitle>{title}</DialogTitle>
            <DialogContent dividers>
                {children}
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={onClose}>
                    Cancel
                </Button>
                <LoadingButton
                    color={confirmColor}
                    loading={confirmInProgress}
                    disabled={confirmDisabled}
                    onClick={onConfirm}
                >
                    {confirmLabel}
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default ConfirmationDialog;
