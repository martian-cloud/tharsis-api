import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import React from "react";

interface Props {
    title: string;
    children: React.ReactNode;
    confirmLabel: string;
    cancelLabel?: string;
    confirmDisabled?: boolean;
    // Defaults to error, since most confirmations guard something destructive. success is for a
    // confirmation whose action is the positive outcome (approving a run gate), and primary for one
    // that is neither — a decision the dialog records rather than a change it destroys.
    confirmColor?: 'error' | 'primary' | 'success';
    confirmInProgress: boolean;
    // When set, the confirm button is a link that opens this URL in a new tab (noopener, noreferrer).
    // A native link click isn't subject to popup blocking the way window.open is, and window.open
    // with noopener returns null whether or not it was blocked, so a blocked open can't be detected.
    confirmHref?: string;
    onConfirm: () => void;
    onClose: () => void;
    maxWidth?: 'xs' | 'sm' | 'md';
}

function ConfirmationDialog({ title, children, confirmLabel, cancelLabel, confirmDisabled, confirmColor = 'error', confirmInProgress, confirmHref, onClose, onConfirm, maxWidth = 'xs', ...other }: Props) {
    return (
        <Dialog
            maxWidth={maxWidth}
            fullWidth
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
                    {cancelLabel || 'Cancel'}
                </Button>
                <Button
                    color={confirmColor}
                    loading={confirmInProgress}
                    disabled={confirmDisabled}
                    {...(confirmHref && { href: confirmHref, target: '_blank', rel: 'noopener noreferrer' })}
                    onClick={onConfirm}
                >
                    {confirmLabel}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

export default ConfirmationDialog;
