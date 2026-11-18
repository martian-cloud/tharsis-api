import CompareArrowsIcon from '@mui/icons-material/CompareArrows';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutlined';
import { Alert, Box, Button, CircularProgress, Paper, Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Suspense, useMemo, useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import ConfirmationDialog from '../common/ConfirmationDialog';
import { MutationError } from '../common/error';
import Timestamp from '../common/Timestamp';
import Link from '../routes/Link';
import { WorkspaceDetailsDriftDetectionFragment_workspace$key } from './__generated__/WorkspaceDetailsDriftDetectionFragment_workspace.graphql';
import { WorkspaceDetailsDriftDetectionMutation } from './__generated__/WorkspaceDetailsDriftDetectionMutation.graphql';
import WorkspaceDetailsDriftViewer from './WorkspaceDetailsDriftViewer';

interface Props {
    fragmentRef: WorkspaceDetailsDriftDetectionFragment_workspace$key;
}

const DriftDetectionButton = ({ onClick }: { onClick: () => void }) => (
    <Button
        variant="outlined"
        color="secondary"
        size="small"
        onClick={onClick}
    >
        Run Drift Detection
    </Button>
);

const StatusMessage = ({
    isInProgress,
    lastAttemptFailed,
    timestamp,
    runId,
    workspacePath
}: {
    isInProgress: boolean,
    lastAttemptFailed: boolean,
    timestamp: string,
    runId?: string,
    workspacePath: string
}) => (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center' }}>
        <Typography color="textSecondary" sx={{
            mr: 1
        }}>
            {isInProgress ? 'Drift detection started' : lastAttemptFailed ? 'Drift detection last attempted' : 'Drift last checked'}{' '}
            <Timestamp
                component="span"
                timestamp={timestamp}
            />
        </Typography>
        {!isInProgress && runId && !lastAttemptFailed && (
            <Link
                to={`/groups/${workspacePath}/-/runs/${runId}`}
                color="textSecondary"
                underline="hover"
            >
                (view assessment run)
            </Link>
        )}
    </Box>
);


function WorkspaceDetailsDriftDetection({ fragmentRef }: Props) {
    const [showConfirmationDialog, setShowConfirmationDialog] = useState(false);
    const [error, setError] = useState<MutationError>();

    const data = useFragment<WorkspaceDetailsDriftDetectionFragment_workspace$key>(
        graphql`
            fragment WorkspaceDetailsDriftDetectionFragment_workspace on Workspace {
                id
                fullPath
                assessment {
                    hasDrift
                    startedAt
                    completedAt
                    errorMessage
                    run {
                        id
                        status
                    }
                }
            }
        `, fragmentRef
    );

    const [commit, isInFlight] = useMutation<WorkspaceDetailsDriftDetectionMutation>(
        graphql`
            mutation WorkspaceDetailsDriftDetectionMutation($input: AssessWorkspaceInput!) {
                assessWorkspace(input: $input) {
                    run {
                        id
                        workspace {
                            fullPath
                            ...WorkspaceDetailsDriftDetectionFragment_workspace
                        }
                    }
                    problems {
                        message
                        field
                        type
                    }
                }
            }
        `
    );

    const onRunDriftDetectionDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commit({
                variables: {
                    input: {
                        workspacePath: data.fullPath
                    }
                },
                onCompleted: data => {
                    if (data.assessWorkspace.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.assessWorkspace.problems.map(problem => problem.message).join('; ')
                        })
                    }
                    else if (!data.assessWorkspace.run) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        })
                    }
                    else {
                        setShowConfirmationDialog(false);
                    }
                }, onError: error => {
                    setError({
                        severity: 'error',
                        message: error.message
                    })
                }
            })
        }
        setShowConfirmationDialog(false);
    };

    const isAssessmentInProgress = useMemo(() => {
        return data.assessment?.startedAt && !data.assessment?.completedAt;
    }, [data.assessment]);

    // When the assessment run errored, no valid drift result was produced (the backend clears
    // the drift verdict), so surface the failure instead of showing a misleading "no drift" state.
    const assessmentRunFailed = useMemo(() => {
        return data.assessment?.run?.status === 'errored';
    }, [data.assessment]);

    // The most recent attempt could not even create an assessment run: the module it was built
    // from is gone, its registry credential expired, or a managed identity rule rejects a
    // system-initiated run. The reason is surfaced alongside any earlier result rather than in
    // place of it.
    const assessmentBlockedReason = useMemo(() => {
        return isAssessmentInProgress ? null : data.assessment?.errorMessage;
    }, [data.assessment, isAssessmentInProgress]);

    // A completed assessment links the run that produced its result, and an attempt that could not
    // create a run leaves any earlier link in place. So a recorded failure with no linked run means
    // no check has ever completed for this workspace, and the stored drift flag is just its default
    // rather than a result. Showing "No drift detected" would claim a clean result nobody computed.
    const assessmentHasNoResult = useMemo(() => {
        return !!assessmentBlockedReason && !data.assessment?.run;
    }, [assessmentBlockedReason, data.assessment]);

    return (
        <Box>
            {error && (
                <Alert sx={{ mb: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>
            )}

            {assessmentBlockedReason && (
                <Alert sx={{ mb: 2 }} severity="warning">
                    <Typography variant="body2" gutterBottom>
                        {assessmentHasNoResult
                            ? 'Drift detection could not run for this workspace, so its drift status is unknown.'
                            : assessmentRunFailed
                                ? 'Drift detection could not run for this workspace, and the status below is from a drift detection run that itself failed.'
                                : 'Drift detection could not run for this workspace, so any status below is from the last successful check.'}
                    </Typography>
                    <Typography variant="body2" sx={{ wordBreak: 'break-word' }}>
                        {assessmentBlockedReason}
                    </Typography>
                </Alert>
            )}

            {data.assessment && (
                <Box sx={{ mb: 2, display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, justifyContent: 'space-between', alignItems: { sm: 'center' }, gap: 2 }}>
                    <Stack direction="row" spacing={2}>
                        <CompareArrowsIcon />
                        <StatusMessage
                            isInProgress={isAssessmentInProgress}
                            lastAttemptFailed={!!assessmentBlockedReason}
                            timestamp={isAssessmentInProgress ? data.assessment.startedAt : data.assessment.completedAt}
                            runId={data.assessment.run?.id}
                            workspacePath={data.fullPath}
                        />
                    </Stack>
                    {!isAssessmentInProgress && (
                        <Box sx={{ alignSelf: { xs: 'center', sm: 'auto' } }}>
                            <DriftDetectionButton
                                onClick={() => setShowConfirmationDialog(true)}
                            />
                        </Box>
                    )}
                </Box>
            )}

            {isAssessmentInProgress && (
                <Paper variant="outlined" sx={{ marginTop: 2, display: "flex", justifyContent: "center" }}>
                    <Box
                        sx={{
                            padding: 4,
                            display: "flex",
                            flexDirection: "column",
                            justifyContent: "center",
                            alignItems: "center"
                        }}>
                        <CircularProgress size={24} sx={{ mb: 2 }} />
                        <Typography color="textSecondary" align="center">
                            Drift detection is in progress
                        </Typography>
                    </Box>
                </Paper>
            )}

            {data.assessment && !isAssessmentInProgress && assessmentRunFailed && (
                <Paper variant="outlined" sx={{ marginTop: 2, display: "flex", justifyContent: "center" }}>
                    <Box
                        sx={{
                            padding: 4,
                            display: "flex",
                            flexDirection: "column",
                            justifyContent: "center",
                            alignItems: "center"
                        }}>
                        <ErrorOutlineIcon color="error" sx={{ mb: 1 }} />
                        <Typography color="error" align="center">
                            The most recent drift detection run failed, so drift status could not be determined
                        </Typography>
                        {data.assessment.run && (
                            <Link
                                to={`/groups/${data.fullPath}/-/runs/${data.assessment.run.id}`}
                                color="textSecondary"
                                underline="hover"
                                sx={{ mt: 1 }}
                            >
                                View failed run
                            </Link>
                        )}
                    </Box>
                </Paper>
            )}

            {data.assessment && !isAssessmentInProgress && !assessmentRunFailed && data.assessment.hasDrift && (
                <Suspense
                    fallback={
                        <Box
                            sx={{
                                minHeight: 400,
                                width: '100%',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center'
                            }}
                        >
                            <CircularProgress />
                        </Box>
                    }
                >
                    <WorkspaceDetailsDriftViewer workspaceId={data.id} />
                </Suspense>
            )}

            {data.assessment && !isAssessmentInProgress && !assessmentRunFailed && !assessmentHasNoResult && !data.assessment.hasDrift && (
                <Paper variant="outlined" sx={{ marginTop: 2, display: "flex", justifyContent: "center" }}>
                    <Box
                        sx={{
                            padding: 4,
                            flexDirection: "column",
                            justifyContent: "center",
                            alignItems: "center"
                        }}>
                        <Typography color="textSecondary">
                            No drift detected for this workspace
                        </Typography>
                    </Box>
                </Paper>
            )}

            {!data.assessment && (
                <Paper variant="outlined" sx={{ marginTop: 4, display: "flex", justifyContent: "center" }}>
                    <Box
                        sx={{
                            padding: 4,
                            display: "flex",
                            flexDirection: "column",
                            justifyContent: "center",
                            alignItems: "center"
                        }}>
                        <Typography variant="h6">
                            This workspace has not been checked for drift
                        </Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            Create a drift detection run to identify any potential drift in your workspace
                        </Typography>
                        <DriftDetectionButton
                            onClick={() => setShowConfirmationDialog(true)}
                        />
                    </Box>
                </Paper>
            )}

            {showConfirmationDialog && (
                <ConfirmationDialog
                    title="Run Drift Detection"
                    confirmLabel="Confirm"
                    confirmColor="primary"
                    confirmInProgress={isInFlight}
                    onConfirm={() => onRunDriftDetectionDialogClosed(true)}
                    onClose={() => onRunDriftDetectionDialogClosed()}
                >
                    Are you sure you want to create a drift detection run?
                </ConfirmationDialog>
            )}
        </Box>
    );
}

export default WorkspaceDetailsDriftDetection;
