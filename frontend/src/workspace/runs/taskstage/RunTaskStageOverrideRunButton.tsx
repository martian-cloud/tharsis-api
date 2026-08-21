import { LoadingButton } from '@mui/lab';
import { Alert, AlertTitle, Stack, TextField, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import ConfirmationDialog from '../../../common/ConfirmationDialog';
import { MutationError } from '../../../common/error';
import { RunTaskStageOverrideRunButtonMutation } from './__generated__/RunTaskStageOverrideRunButtonMutation.graphql';

interface Props {
    // The gate blocking the check. This button is for the gate nobody can approve — one whose failed
    // policies declared no approvers, so it has no approval rules and an override is the only way
    // past it. A gate with rules is decided through RunTaskStageRunGateDecisionButtons instead.
    gateId: string
    disabled?: boolean
    onError: (error: MutationError) => void
}

// RunTaskStageOverrideRunButton lets a permitted user override a run that is stalled at the
// awaiting_override policy stage state (a soft-fail gate could not be satisfied). It
// mirrors ForceCancelRunButton: a LoadingButton opening a ConfirmationDialog. The
// button is disabled unless the caller determines an override is currently possible.
function RunTaskStageOverrideRunButton(props: Props) {
    const { gateId, disabled, onError } = props;
    const [openDialog, setOpenDialog] = useState(false);
    const [comment, setComment] = useState('');

    const [commitOverrideRun, commitOverrideRunInFlight] = useMutation<RunTaskStageOverrideRunButtonMutation>(graphql`
        mutation RunTaskStageOverrideRunButtonMutation($input: OverrideRunGateInput!) {
            overrideRunGate(input: $input) {
                runGate {
                    id
                    status
                    overriddenBy
                    overrideComment
                    metadata {
                        updatedAt
                    }
                    # The check and the run both move when the gate clears, so they are selected
                    # through the gate to update the page from the store without a refetch.
                    run {
                        id
                        status
                        taskStages {
                            stageName
                            status
                            policyChecks {
                                status
                            }
                        }
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

    const overrideRun = (confirm?: boolean) => {
        if (!confirm) {
            setOpenDialog(false);
            setComment('');
            return;
        }
        commitOverrideRun({
            variables: {
                input: {
                    gateId,
                    comment: comment || undefined
                },
            },
            onCompleted: data => {
                setOpenDialog(false);
                setComment('');
                if (data.overrideRunGate.problems.length) {
                    onError({
                        severity: 'warning',
                        message: data.overrideRunGate.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                setOpenDialog(false);
                setComment('');
                onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        });
    };

    return (
        <Stack direction="row" spacing={2}>
            <LoadingButton
                loading={commitOverrideRunInFlight}
                disabled={disabled}
                size="medium"
                variant="outlined"
                color="warning"
                onClick={() => setOpenDialog(true)}
            >
                Override
            </LoadingButton>
            {openDialog && (
                <ConfirmationDialog
                    title="Override Run"
                    maxWidth="sm"
                    confirmLabel="Override"
                    confirmInProgress={commitOverrideRunInFlight}
                    onConfirm={() => overrideRun(true)}
                    onClose={() => overrideRun()}
                >
                    <Alert sx={{ mb: 2 }} severity="warning">
                        <AlertTitle>Warning</AlertTitle>
                        Overriding allows this run to proceed past a policy soft-fail that could not be satisfied by the required approvals.
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
            )}
        </Stack>
    );
}

export default RunTaskStageOverrideRunButton;
