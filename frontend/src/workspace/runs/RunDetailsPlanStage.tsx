import { Alert, Button, CircularProgress, Paper, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import React, { Suspense, useMemo, useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import Link from '../../routes/Link';
import ForceCancelRunAlert from './ForceCancelRunAlert';
import JobLogs from './JobLogs';
import NoRunnerAlert from './NoRunnerAlert';
import RunDetailsErrorSummary from './RunDetailsErrorSummary';
import RunDetailsPlanSummary from './RunDetailsPlanSummary';
import RunJobDialog from './RunJobDialog';
import RunDetailsStageHeader from './RunDetailsStageHeader';
import RunDetailsStageJobsCard from './RunDetailsStageJobsCard';
import RunDetailsStageLogsPending from './RunDetailsStageLogsPending';
import RunDetailsStageStatusCard from './RunDetailsStageStatusCard';
import RunDetailsStageTabEmptyState from './RunDetailsStageTabEmptyState';
import RunStageStatusTypes from './RunStageStatusTypes';
import RunVariables from './RunVariables';
import { RunDetailsPlanStageApplyRunMutation } from './__generated__/RunDetailsPlanStageApplyRunMutation.graphql';
import { RunDetailsPlanStageCancelRunMutation } from './__generated__/RunDetailsPlanStageCancelRunMutation.graphql';
import { RunDetailsPlanStageDiscardRunMutation } from './__generated__/RunDetailsPlanStageDiscardRunMutation.graphql';
import { RunDetailsPlanStageUndiscardRunMutation } from './__generated__/RunDetailsPlanStageUndiscardRunMutation.graphql';
import { RunDetailsPlanStageRetryRunNodeMutation } from './__generated__/RunDetailsPlanStageRetryRunNodeMutation.graphql';
import { RunDetailsPlanStageFragment_plan$key } from './__generated__/RunDetailsPlanStageFragment_plan.graphql';
import RunDetailsPlanDiffViewer, { MaxDiffSize } from './plandiff/RunDetailsPlanDiffViewer';

interface Props {
    fragmentRef: RunDetailsPlanStageFragment_plan$key
    onError: (error: MutationError) => void
}

// getPlanDescription returns the contextual description shown in the status card. The
// page keeps one fixed layout across all plan states; only this text (and the card's
// actions) change with the state.
function getPlanDescription(planStatus: string, jobStatus?: string): string | undefined {
    if (jobStatus === 'pending') {
        return 'A runner has picked up the plan job and it will start shortly';
    }
    switch (planStatus) {
        case 'pending':
            return 'The plan is waiting to be queued and will start once the workspace is available to run it';
        case 'queued':
            return 'The plan job is queued and is waiting to be claimed by a runner';
        case 'canceled':
            return 'The plan was canceled before it completed';
        default:
            return undefined;
    }
}

function RunDetailsPlanStage(props: Props) {
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const tab = searchParams.get('tab') ?? 'logs';
    const jobId = searchParams.get('jobId');
    const [jobDialogOpen, setJobDialogOpen] = useState(false);

    const data = useFragment<RunDetailsPlanStageFragment_plan$key>(
        graphql`
        fragment RunDetailsPlanStageFragment_plan on Run
        {
            id
            status
            createdBy
            plan {
                metadata {
                    createdAt
                }
                status
                errorMessage
                hasChanges
                diffSize
                currentJob {
                  id
                  status
                  cancelRequested
                  timestamps {
                    queuedAt
                    pendingAt
                    runningAt
                    finishedAt
                  }
                  ...NoRunnerAlertFragment_job
                }
                jobs(first: 0) {
                  totalCount
                }
                ...RunDetailsPlanSummaryFragment_plan
            }
            apply {
                status
            }
            ...RunVariablesFragment_variables
            ...ForceCancelRunAlertFragment_run
        }
      `, props.fragmentRef);

    const [commitApplyRun, commitApplyRunInFlight] = useMutation<RunDetailsPlanStageApplyRunMutation>(graphql`
        mutation RunDetailsPlanStageApplyRunMutation($input: ApplyRunInput!) {
            applyRun(input: $input) {
                run {
                    ...RunDetailsPlanStageFragment_plan
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const applyRun = () => {
        commitApplyRun({
            variables: {
                input: {
                    runId: data.id,
                    comment: ''
                },
            },
            onCompleted: data => {
                if (data.applyRun.problems.length) {
                    props.onError({
                        severity: 'warning',
                        message: data.applyRun.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                props.onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        })
    }

    const [commitCancelRun, commitCancelRunInFlight] = useMutation<RunDetailsPlanStageCancelRunMutation>(graphql`
        mutation RunDetailsPlanStageCancelRunMutation($input: CancelRunInput!) {
            cancelRun(input: $input) {
                run {
                    ...RunDetailsPlanStageFragment_plan
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const cancelRun = () => {
        commitCancelRun({
            variables: {
                input: {
                    runId: data.id
                },
            },
            onCompleted: data => {
                if (data.cancelRun.problems.length) {
                    props.onError({
                        severity: 'warning',
                        message: data.cancelRun.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                props.onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        })
    }

    const [commitDiscardRun, commitDiscardRunInFlight] = useMutation<RunDetailsPlanStageDiscardRunMutation>(graphql`
        mutation RunDetailsPlanStageDiscardRunMutation($input: DiscardRunInput!) {
            discardRun(input: $input) {
                run {
                    ...RunDetailsPlanStageFragment_plan
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const discardRun = () => {
        commitDiscardRun({
            variables: {
                input: {
                    runId: data.id
                },
            },
            onCompleted: data => {
                if (data.discardRun.problems.length) {
                    props.onError({
                        severity: 'warning',
                        message: data.discardRun.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                props.onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        })
    }

    const [commitUndiscardRun, commitUndiscardRunInFlight] = useMutation<RunDetailsPlanStageUndiscardRunMutation>(graphql`
        mutation RunDetailsPlanStageUndiscardRunMutation($input: UndiscardRunInput!) {
            undiscardRun(input: $input) {
                run {
                    ...RunDetailsPlanStageFragment_plan
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const undiscardRun = () => {
        commitUndiscardRun({
            variables: {
                input: {
                    runId: data.id
                },
            },
            onCompleted: data => {
                if (data.undiscardRun.problems.length) {
                    props.onError({
                        severity: 'warning',
                        message: data.undiscardRun.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                props.onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        })
    }

    const [commitRetryRunNode, commitRetryRunNodeInFlight] = useMutation<RunDetailsPlanStageRetryRunNodeMutation>(graphql`
        mutation RunDetailsPlanStageRetryRunNodeMutation($input: RetryRunNodeInput!) {
            retryRunNode(input: $input) {
                run {
                    ...RunDetailsPlanStageFragment_plan
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const retryRunNode = () => {
        commitRetryRunNode({
            variables: {
                input: {
                    runId: data.id,
                    nodePath: 'plan'
                },
            },
            onCompleted: data => {
                if (data.retryRunNode.problems.length) {
                    props.onError({
                        severity: 'warning',
                        message: data.retryRunNode.problems.map(problem => problem.message).join('; ')
                    });
                }
            },
            onError: error => {
                props.onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        })
    }

    const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
        navigate({
            search: `?tab=${newValue}`
        }, {
            replace: true
        });
    };

    const durationMs = useMemo(() => {
        const timestamps = data.plan.currentJob?.timestamps;
        return timestamps?.runningAt && timestamps?.finishedAt ?
            moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput)) : null;
    }, [data.plan.currentJob]);

    // When no jobId param is set, the logs follow the plan's latest (current) job. When a
    // jobId param is set (e.g. selected from the jobs dialog), the view stays pinned to that
    // job until the param is cleared or changed — even as a new current job appears (e.g.
    // after a retry).
    const effectiveJobId = jobId ?? data.plan.currentJob?.id;
    const viewingEarlierJob = !!jobId && !!data.plan.currentJob && jobId !== data.plan.currentJob.id;

    // Logs only exist once the plan is running or has reached a final state, so the
    // log viewer is hidden before then. A job pinned via the jobs dialog is always
    // viewable (it has already run). While the job is pending (a runner claimed it
    // and is preparing to run) the logs area shows the launch animation instead.
    const jobPending = data.plan.currentJob?.status === 'pending';
    const logsAvailable = !!jobId || ['running', 'finished', 'errored', 'canceled'].includes(data.plan.status);

    const planStatusType = RunStageStatusTypes[data.plan.status] ?? { label: 'unknown', color: 'runStatus.unknown' };
    const StatusIcon = planStatusType.icon;

    const maxDiffSizeExceeded = useMemo(() => data.plan.diffSize > MaxDiffSize, [data.plan.diffSize]);

    const description = getPlanDescription(data.plan.status, data.plan.currentJob?.status);

    // Status card actions: the action set varies with the state but always lives in the
    // same place. The conditions are mutually exclusive.
    let statusCardActions: React.ReactNode;
    if (data.status === 'planned' && data.apply && data.apply.status === 'created' && data.plan.hasChanges) {
        statusCardActions = <React.Fragment>
            <Button loading={commitDiscardRunInFlight} variant="outlined" size="medium" color="info" onClick={discardRun}>
                Discard
            </Button>
            <Button loading={commitApplyRunInFlight} variant="outlined" size="medium" onClick={applyRun}>
                Start Apply
            </Button>
        </React.Fragment>;
    } else if (data.status === 'discarded') {
        statusCardActions = <Button loading={commitUndiscardRunInFlight} variant="outlined" size="medium" color="info" onClick={undiscardRun}>
            Undiscard
        </Button>;
    } else if (data.plan.status === 'errored' || data.plan.status === 'canceled') {
        statusCardActions = <Button loading={commitRetryRunNodeInFlight} variant="outlined" size="medium" color="warning" onClick={retryRunNode}>
            Retry
        </Button>;
    } else if (['pending', 'queued', 'running'].includes(data.plan.status) && !data.plan.currentJob?.cancelRequested) {
        statusCardActions = <Button loading={commitCancelRunInFlight} variant="outlined" size="medium" color="error" onClick={cancelRun}>
            Cancel
        </Button>;
    }

    // Status card expansion content (below the divider): the error summary for a failed
    // plan, or the plan summary / no-changes note for a finished one.
    let statusCardContent: React.ReactNode;
    if (data.plan.status === 'errored' && data.plan.errorMessage) {
        statusCardContent = <RunDetailsErrorSummary errorMessage={data.plan.errorMessage} ml={-2} mr={-2} mb={-2} />;
    } else if (data.plan.status === 'finished') {
        statusCardContent = data.plan.hasChanges ? <React.Fragment>
            <RunDetailsPlanSummary fragmentRef={data.plan} ml={-2} mr={-2} completed={false} />
            <Box mt={2}>
                <Link color="secondary" to={'?tab=changes'}>View changes</Link>
            </Box>
        </React.Fragment> : <Typography mt={2} variant="body2">
            This plan does not contain any changes to apply
        </Typography>;
    }

    return (
        <Box>
            {data.plan.currentJob?.cancelRequested && data.plan.status !== 'canceled' && <ForceCancelRunAlert fragmentRef={data} />}
            {data.plan.currentJob && <NoRunnerAlert fragmentRef={data.plan.currentJob} sx={{ mb: 2 }} />}
            <RunDetailsStageHeader stage="Plan" triggeredAt={data.plan.metadata.createdAt as string} triggeredBy={data.createdBy} />
            <RunDetailsStageStatusCard
                icon={<StatusIcon sx={{ width: 32, height: 32, mr: 2 }} />}
                title={`Plan ${planStatusType.label}`}
                durationMs={durationMs}
                description={description}
                actions={statusCardActions}
            >
                {statusCardContent}
            </RunDetailsStageStatusCard>
            <RunDetailsStageJobsCard stage="plan" totalCount={data.plan.jobs.totalCount} onOpenJobs={() => setJobDialogOpen(true)} />
            <Box>
                <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                    <Tabs value={tab} onChange={onTabChange} variant="scrollable" scrollButtons="auto" allowScrollButtonsMobile>
                        <Tab label="Logs" value="logs" />
                        <Tab label="Variables" value="variables" />
                        <Tab label="Changes" value="changes" />
                    </Tabs>
                </Box>
                {tab === 'logs' && <Box>
                    {viewingEarlierJob && <Alert severity="warning" sx={{ my: 2 }}>
                        Showing logs for an earlier job{' '}
                        <Link color="inherit" to={'?tab=logs'}>(view latest job)</Link>
                    </Alert>}
                    {logsAvailable && effectiveJobId && <React.Fragment>
                        <Suspense fallback={<Box sx={{ minHeight: 120, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><CircularProgress /></Box>}>
                            <JobLogs jobId={effectiveJobId} />
                        </Suspense>
                        {data.plan.hasChanges && data.plan.status === 'finished' && <Box mt={2}>
                            <Link color="secondary" to={'?tab=changes'}>View plan changes</Link>
                        </Box>}
                    </React.Fragment>}
                    {!(logsAvailable && effectiveJobId) && (jobPending ? <RunDetailsStageLogsPending /> : <Box marginTop={2}>
                        <RunDetailsStageTabEmptyState message="Logs will be displayed once the plan has started" />
                    </Box>)}
                </Box>}
                {tab === 'variables' && <Box marginTop={2}>
                    <RunVariables fragmentRef={data} />
                </Box>}
                {tab === 'changes' && <Box marginTop={2}>
                    {maxDiffSizeExceeded && <Paper
                        variant="outlined"
                        sx={{
                            minHeight: 100,
                            display: 'flex',
                            flexDirection: 'column',
                            justifyContent: 'center',
                        }}
                    >
                        <Typography align="center">
                            Plan diff is too large to display. Diff size of {data.plan.diffSize} bytes exceeds the maximum limit of {MaxDiffSize} bytes
                        </Typography>
                    </Paper>}
                    {data.plan.status === 'finished' && !maxDiffSizeExceeded && <Suspense fallback={<Box
                        sx={{
                            minHeight: 400,
                            width: '100%',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center'
                        }}
                    >
                        <CircularProgress />
                    </Box>}>
                        <RunDetailsPlanDiffViewer runId={data.id} />
                    </Suspense>}
                    {data.plan.status !== 'finished' && <RunDetailsStageTabEmptyState
                        message={['pending', 'queued', 'running'].includes(data.plan.status) ?
                            'Changes will be displayed once the plan has completed' : 'This plan does not contain any changes'}
                    />}
                </Box>}
            </Box>
            {jobDialogOpen && <RunJobDialog
                runId={data.id}
                stage="plan"
                onSelectJob={(id) => {
                    navigate({ search: `?tab=logs&jobId=${id}` }, { replace: true });
                    setJobDialogOpen(false);
                }}
                onClose={() => setJobDialogOpen(false)}
            />}
        </Box>
    );
}

export default RunDetailsPlanStage;
