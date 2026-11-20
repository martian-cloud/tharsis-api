import { Button, Dialog, DialogActions, DialogContent, DialogTitle, TextField, Typography } from "@mui/material";
import React, { useState } from "react";
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';

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
    // When set, confirm stays disabled until the user types value exactly.
    typedConfirmation?: { label: string; value: string };
}

function ConfirmationDialog({ title, children, confirmLabel, cancelLabel, confirmDisabled, confirmColor = 'error', confirmInProgress, confirmHref, onClose, onConfirm, maxWidth = 'xs', typedConfirmation, ...other }: Props) {
    const [typed, setTyped] = useState('');
    const [typedFor, setTypedFor] = useState(typedConfirmation?.value);

    // Clear the input whenever the required value changes or the requirement is toggled off and back on.
    if (typedFor !== typedConfirmation?.value) {
        setTypedFor(typedConfirmation?.value);
        setTyped('');
    }

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
                {typedConfirmation && <>
                    <Typography variant="subtitle2">{typedConfirmation.label}</Typography>
                    <SyntaxHighlighter style={prismTheme} customStyle={{ fontSize: 14, marginBottom: 14 }}>{typedConfirmation.value}</SyntaxHighlighter>
                    <TextField
                        autoComplete="off"
                        fullWidth
                        size="small"
                        placeholder={typedConfirmation.value}
                        value={typed}
                        onChange={event => setTyped(event.target.value)}
                        slotProps={{ htmlInput: { 'aria-label': typedConfirmation.label } }}
                    />
                </>}
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={onClose}>
                    {cancelLabel || 'Cancel'}
                </Button>
                <Button
                    color={confirmColor}
                    loading={confirmInProgress}
                    disabled={confirmDisabled || (!!typedConfirmation && typed !== typedConfirmation.value)}
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
