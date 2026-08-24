import { LoadingButton } from '@mui/lab';
import { Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import ConfirmationDialog from '../../../common/ConfirmationDialog';
import { MutationError } from '../../../common/error';
import { RunTaskStageCancelRunButtonMutation } from './__generated__/RunTaskStageCancelRunButtonMutation.graphql';

interface Props {
    runId: string;
    onError: (error: MutationError) => void;
}

// RunTaskStageCancelRunButton cancels the run from the policy stage view, behind a ConfirmationDialog like the
// other destructive run actions here. Cancellation is a run-level operation, not a stage-level one:
// the server cancels whichever phase the run is currently in, which is this stage whenever the stage
// is not yet final, and cascades the run to canceled. The API rejects a cancel once the run is final
// (applied, planned and finished, errored, canceled or discarded), so callers render this only while
// the stage can still be stopped.
function RunTaskStageCancelRunButton({ runId, onError }: Props) {
    const [openDialog, setOpenDialog] = useState(false);

    const [commitCancelRun, commitCancelRunInFlight] = useMutation<RunTaskStageCancelRunButtonMutation>(graphql`
        mutation RunTaskStageCancelRunButtonMutation($input: CancelRunInput!) {
            cancelRun(input: $input) {
                run {
                    ...RunDetailsRunTaskStageFragment_taskStage
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const cancelRun = (confirm?: boolean) => {
        if (!confirm) {
            setOpenDialog(false);
            return;
        }
        commitCancelRun({
            variables: { input: { runId } },
            onCompleted: data => {
                setOpenDialog(false);
                if (data.cancelRun.problems.length) {
                    onError({
                        severity: 'warning',
                        message: data.cancelRun.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                setOpenDialog(false);
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
                loading={commitCancelRunInFlight}
                size="small"
                variant="outlined"
                color="error"
                onClick={() => setOpenDialog(true)}
            >
                Cancel Run
            </LoadingButton>
            {openDialog && (
                <ConfirmationDialog
                    title="Cancel Run"
                    maxWidth="sm"
                    confirmLabel="Cancel Run"
                    cancelLabel="No"
                    confirmInProgress={commitCancelRunInFlight}
                    onConfirm={() => cancelRun(true)}
                    onClose={() => cancelRun()}
                >
                    <Typography gutterBottom>
                        Are you sure you want to cancel this run?
                    </Typography>
                </ConfirmationDialog>
            )}
        </Stack>
    );
}

export default RunTaskStageCancelRunButton;
