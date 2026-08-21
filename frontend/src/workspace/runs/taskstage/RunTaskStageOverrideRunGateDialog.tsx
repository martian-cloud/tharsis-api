import { Alert, AlertTitle, TextField, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import ConfirmationDialog from '../../../common/ConfirmationDialog';
import { MutationError } from '../../../common/error';
import { RunTaskStageOverrideRunGateDialogMutation } from './__generated__/RunTaskStageOverrideRunGateDialogMutation.graphql';

interface Props {
    gateId: string
    onClose: () => void
    onError: (error: MutationError) => void
}

// RunTaskStageOverrideRunGateDialog confirms bypassing a gate's required approvals. It renders no trigger of its
// own and is mounted only while open, because its trigger is a MenuItem: a dialog rendered inside a
// Menu would be unmounted the moment the menu closed.
function RunTaskStageOverrideRunGateDialog({ gateId, onClose, onError }: Props) {
    const [comment, setComment] = useState('');

    const [commitOverrideGate, commitOverrideGateInFlight] = useMutation<RunTaskStageOverrideRunGateDialogMutation>(graphql`
        mutation RunTaskStageOverrideRunGateDialogMutation($input: OverrideRunGateInput!) {
            overrideRunGate(input: $input) {
                runGate {
                    id
                    status
                    # Selected so the panel's "approvals bypassed" banner appears from the store
                    # without a refetch.
                    overriddenBy
                    overrideComment
                    metadata {
                        updatedAt
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const overrideGate = () => {
        commitOverrideGate({
            variables: {
                input: {
                    gateId,
                    comment: comment || undefined
                },
            },
            onCompleted: data => {
                onClose();
                if (data.overrideRunGate.problems.length) {
                    onError({
                        severity: 'warning',
                        message: data.overrideRunGate.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                onClose();
                onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        });
    };

    return (
        <ConfirmationDialog
            title="Override Gate"
            maxWidth="sm"
            confirmLabel="Override"
            confirmInProgress={commitOverrideGateInFlight}
            onConfirm={overrideGate}
            onClose={onClose}
        >
            <Alert sx={{ mb: 2 }} severity="warning">
                <AlertTitle>Warning</AlertTitle>
                Overriding this gate bypasses the required approvals and allows the run to proceed.
            </Alert>
            <Typography variant="subtitle2" gutterBottom>Comment (optional)</Typography>
            <TextField
                autoComplete="off"
                fullWidth
                multiline
                minRows={2}
                size="small"
                placeholder="Reason for overriding"
                value={comment}
                onChange={(e) => setComment(e.target.value)}
            />
        </ConfirmationDialog>
    );
}

export default RunTaskStageOverrideRunGateDialog;
