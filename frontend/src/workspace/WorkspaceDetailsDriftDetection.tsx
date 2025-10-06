import CompareArrowsIcon from '@mui/icons-material/CompareArrows';
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
    timestamp,
    runId,
    workspacePath
}: {
    isInProgress: boolean,
    timestamp: string,
    runId?: string,
    workspacePath: string
}) => (
    <Box display="flex">
        <Typography color="textSecondary" mr={1}>
            {isInProgress ? 'Drift detection started' : 'Drift last checked'}{' '}
            <Timestamp
                component="span"
                timestamp={timestamp}
            />
        </Typography>
        {!isInProgress && runId && (
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
                    run {
                        id
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

    return (
        <Box>
            {error && (
                <Alert sx={{ mb: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>
            )}

            {data.assessment && (
                <Box sx={{ mb: 2 }} display="flex" justifyContent="space-between">
                    <Stack direction="row" spacing={2}>
                        <CompareArrowsIcon />
                        <StatusMessage
                            isInProgress={isAssessmentInProgress}
                            timestamp={isAssessmentInProgress ? data.assessment.startedAt : data.assessment.completedAt}
                            runId={data.assessment.run?.id}
                            workspacePath={data.fullPath}
                        />
                    </Stack>
                    {!isAssessmentInProgress && (
                        <Box>
                            <DriftDetectionButton
                                onClick={() => setShowConfirmationDialog(true)}
                            />
                        </Box>
                    )}
                </Box>
            )}

            {isAssessmentInProgress && (
                <Paper variant="outlined" sx={{ marginTop: 2, display: "flex", justifyContent: "center" }}>
                    <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                        <CircularProgress size={24} sx={{ mb: 2 }} />
                        <Typography color="textSecondary" align="center">
                            Drift detection is in progress
                        </Typography>
                    </Box>
                </Paper>
            )}

            {data.assessment && !isAssessmentInProgress && data.assessment.hasDrift && (
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

            {data.assessment && !isAssessmentInProgress && !data.assessment.hasDrift && (
                <Paper variant="outlined" sx={{ marginTop: 2, display: "flex", justifyContent: "center" }}>
                    <Box padding={4} flexDirection="column" justifyContent="center" alignItems="center">
                        <Typography color="textSecondary">
                            No drift detected for this workspace
                        </Typography>
                    </Box>
                </Paper>
            )}

            {!data.assessment && (
                <Paper variant="outlined" sx={{ marginTop: 4, display: "flex", justifyContent: "center" }}>
                    <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
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
                    message="Are you sure you want to create a drift detection run?"
                    confirmButtonLabel="Confirm"
                    opInProgress={isInFlight}
                    onConfirm={() => onRunDriftDetectionDialogClosed(true)}
                    onClose={() => onRunDriftDetectionDialogClosed()}
                />
            )}
        </Box>
    );
}

export default WorkspaceDetailsDriftDetection;
