import { Alert, Button, CircularProgress, Typography } from '@mui/material';
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
import CheckResultsPanel from './CheckResultsPanel';
import { RunDetailsApplyStageApplyRunMutation } from './__generated__/RunDetailsApplyStageApplyRunMutation.graphql';
import { RunDetailsApplyStageCancelRunMutation } from './__generated__/RunDetailsApplyStageCancelRunMutation.graphql';
import { RunDetailsApplyStageFragment_apply$key } from './__generated__/RunDetailsApplyStageFragment_apply.graphql';
import { RunDetailsApplyStageRetryRunNodeMutation } from './__generated__/RunDetailsApplyStageRetryRunNodeMutation.graphql';

interface Props {
    fragmentRef: RunDetailsApplyStageFragment_apply$key
    onError: (error: MutationError) => void
}

// RUN_FINAL_STATUSES are the run statuses in which a never-started apply will never run.
const RUN_FINAL_STATUSES = ['errored', 'canceled', 'discarded', 'planned_and_finished'];

// getApplyDescription returns the contextual description shown in the status card. The
// page keeps one fixed layout across all apply states; only this text (and the card's
// actions) change with the state. A never-started apply (created) and a skipped apply
// both describe themselves in terms of the run status: the backend skips a created
// apply when the run reaches a final state (and un-skips it on a plan retry), so
// keying both off the run status also renders sensible text during those transitions.
function getApplyDescription(applyStatus: string, runStatus: string, jobStatus?: string): string | undefined {
    if (jobStatus === 'pending') {
        return 'A runner has picked up the apply job and it will start shortly';
    }
    if (applyStatus === 'created' || applyStatus === 'skipped') {
        switch (runStatus) {
            case 'pending':
            case 'queuing':
            case 'plan_queued':
            case 'planning':
                return 'The apply can be started after all previous stages have completed';
            case 'planned':
                return 'All previous stages have completed and the apply is ready to be started';
            case 'planned_and_finished':
                return 'The apply was skipped because the plan did not contain any changes';
            case 'errored':
                return 'The apply was not started because the run failed';
            case 'canceled':
                return 'The apply was not started because the run was canceled';
            case 'discarded':
                return 'The apply was not started because the run was discarded';
            default:
                return undefined;
        }
    }
    switch (applyStatus) {
        case 'pending':
            return 'The apply is waiting to be queued and will start once the workspace is available to run it';
        case 'queued':
            return 'The apply job is queued and is waiting to be claimed by a runner';
        case 'canceled':
            return 'The apply was canceled before it completed';
        default:
            return undefined;
    }
}

function RunDetailsApplyStage(props: Props) {
    const [jobDialogOpen, setJobDialogOpen] = useState(false);
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const jobId = searchParams.get('jobId');

    // The tab lives in the URL like the plan stage's. Sanitize it: a stale value
    // carried over from the plan page (e.g. ?tab=changes) falls back to logs.
    const tabParam = searchParams.get('tab');
    const tab = tabParam === 'variables' ? 'variables' : 'logs';

    const data = useFragment<RunDetailsApplyStageFragment_apply$key>(
        graphql`
        fragment RunDetailsApplyStageFragment_apply on Run
        {
            id
            status
            plan {
                status
                ...RunDetailsPlanSummaryFragment_plan
            }
            apply {
                metadata {
                    createdAt
                }
                status
                errorMessage
                triggeredBy
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
            }
            ...RunVariablesFragment_variables
            ...ForceCancelRunAlertFragment_run
            stateVersion {
                inventory {
                    checkResults {
                        ...CheckResultsPanelFragment_checkResult
                    }
                }
            }
        }
      `, props.fragmentRef)

    const [commitApplyRun, commitApplyRunInFlight] = useMutation<RunDetailsApplyStageApplyRunMutation>(graphql`
        mutation RunDetailsApplyStageApplyRunMutation($input: ApplyRunInput!) {
            applyRun(input: $input) {
                run {
                    ...RunDetailsApplyStageFragment_apply
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

    const [commitCancelRun, commitCancelRunInFlight] = useMutation<RunDetailsApplyStageCancelRunMutation>(graphql`
        mutation RunDetailsApplyStageCancelRunMutation($input: CancelRunInput!) {
            cancelRun(input: $input) {
                run {
                    ...RunDetailsApplyStageFragment_apply
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

    const [commitRetryRunNode, commitRetryRunNodeInFlight] = useMutation<RunDetailsApplyStageRetryRunNodeMutation>(graphql`
        mutation RunDetailsApplyStageRetryRunNodeMutation($input: RetryRunNodeInput!) {
            retryRunNode(input: $input) {
                run {
                    ...RunDetailsApplyStageFragment_apply
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
                    nodePath: 'apply'
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
        const timestamps = data.apply?.currentJob?.timestamps;
        return timestamps?.runningAt && timestamps?.finishedAt ?
            moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput)) : null;
    }, [data.apply?.currentJob]);

    const apply = data.apply;
    if (!apply) {
        // Speculative and assessment runs have no apply stage; the sidebar never links
        // here, so this only renders on direct URL navigation.
        return (
            <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Typography color="textSecondary">This run does not have an apply stage</Typography>
            </Box>
        );
    }

    // When no jobId param is set, the logs follow the apply's latest (current) job. When a
    // jobId param is set (e.g. selected from the jobs dialog), the view stays pinned to that
    // job until the param is cleared or changed — even as a new current job appears (e.g.
    // after a retry).
    const effectiveJobId = jobId ?? apply.currentJob?.id;
    const viewingEarlierJob = !!jobId && !!apply.currentJob && jobId !== apply.currentJob.id;

    // Logs only exist once the apply is running or has reached a final state, so the
    // log viewer is hidden before then. A job pinned via the jobs dialog is always
    // viewable (it has already run). While the job is pending (a runner claimed it
    // and is preparing to run) the logs area shows the launch animation instead.
    const jobPending = apply.currentJob?.status === 'pending';
    const logsAvailable = !!jobId || ['running', 'finished', 'errored', 'canceled'].includes(apply.status);

    const applyStatusType = RunStageStatusTypes[apply.status] ?? { label: 'unknown', color: 'runStatus.unknown' };
    const StatusIcon = applyStatusType.icon;

    const description = getApplyDescription(apply.status, data.status, apply.currentJob?.status);

    // Status card actions: the action set varies with the state but always lives in the
    // same place. The conditions are mutually exclusive.
    let statusCardActions: React.ReactNode;
    if (data.status === 'planned' && apply.status === 'created') {
        statusCardActions = <Button loading={commitApplyRunInFlight} variant="outlined" size="medium" onClick={applyRun}>
            Start Apply
        </Button>;
    } else if (apply.status === 'errored' || apply.status === 'canceled') {
        statusCardActions = <Button loading={commitRetryRunNodeInFlight} variant="outlined" size="medium" color="warning" onClick={retryRunNode}>
            Retry
        </Button>;
    } else if (['pending', 'queued', 'running'].includes(apply.status) && !apply.currentJob?.cancelRequested) {
        statusCardActions = <Button loading={commitCancelRunInFlight} variant="outlined" size="medium" color="error" onClick={cancelRun}>
            Cancel
        </Button>;
    }

    // Status card expansion content (below the divider): the error summary for a failed
    // apply, or the applied changes summary for a finished one.
    let statusCardContent: React.ReactNode;
    if (apply.status === 'errored' && apply.errorMessage) {
        statusCardContent = <RunDetailsErrorSummary errorMessage={apply.errorMessage} ml={-2} mr={-2} mb={-2} />;
    } else if (apply.status === 'finished') {
        statusCardContent = <React.Fragment>
            <RunDetailsPlanSummary fragmentRef={data.plan} ml={-2} mr={-2} completed={true} />
            <Box mt={2}>
                <Link color="secondary" to={'../plan?tab=changes'}>View changes</Link>
            </Box>
        </React.Fragment>;
    }

    // A never-started apply on a finished run will never produce logs; otherwise the
    // logs simply haven't started yet.
    const logsEmptyMessage = (apply.status === 'skipped' || (apply.status === 'created' && RUN_FINAL_STATUSES.includes(data.status))) ?
        'No logs are available because the apply was not run' :
        'Logs will be displayed once the apply has started';

    return (
        <Box>
            {apply.currentJob?.cancelRequested && apply.status !== 'canceled' && <ForceCancelRunAlert fragmentRef={data} />}
            {apply.currentJob && <NoRunnerAlert fragmentRef={apply.currentJob} sx={{ mb: 2 }} />}
            <RunDetailsStageHeader stage="Apply" triggeredAt={apply.metadata.createdAt as string} triggeredBy={apply.triggeredBy} />
            <RunDetailsStageStatusCard
                icon={<StatusIcon sx={{ width: 32, height: 32, mr: 2 }} />}
                title={`Apply ${applyStatusType.label}`}
                durationMs={durationMs}
                description={description}
                actions={statusCardActions}
            >
                {statusCardContent}
            </RunDetailsStageStatusCard>
            {apply.status === 'finished' && data.stateVersion && data.stateVersion.inventory.checkResults.length > 0 && <CheckResultsPanel fragmentRefs={data.stateVersion.inventory.checkResults} />}
            <RunDetailsStageJobsCard stage="apply" totalCount={apply.jobs.totalCount} onOpenJobs={() => setJobDialogOpen(true)} />
            <Box>
                <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                    <Tabs value={tab} onChange={onTabChange} variant="scrollable" scrollButtons="auto" allowScrollButtonsMobile>
                        <Tab label="Logs" value="logs" />
                        <Tab label="Variables" value="variables" />
                    </Tabs>
                </Box>
                {tab === 'logs' && <Box>
                    {viewingEarlierJob && <Alert severity="warning" sx={{ my: 2 }}>
                        Showing logs for an earlier job{' '}
                        <Link color="inherit" to={'?tab=logs'}>(view latest job)</Link>
                    </Alert>}
                    {logsAvailable && effectiveJobId && <Suspense fallback={<Box sx={{ minHeight: 120, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><CircularProgress /></Box>}>
                        <JobLogs jobId={effectiveJobId} />
                    </Suspense>}
                    {!(logsAvailable && effectiveJobId) && (jobPending ? <RunDetailsStageLogsPending /> : <Box marginTop={2}>
                        <RunDetailsStageTabEmptyState message={logsEmptyMessage} />
                    </Box>)}
                </Box>}
                {tab === 'variables' && <Box marginTop={2}>
                    <RunVariables fragmentRef={data} />
                </Box>}
            </Box>
            {jobDialogOpen && <RunJobDialog
                runId={data.id}
                stage="apply"
                onSelectJob={(id) => {
                    navigate({ search: `?tab=logs&jobId=${id}` }, { replace: true });
                    setJobDialogOpen(false);
                }}
                onClose={() => setJobDialogOpen(false)}
            />}
        </Box>
    );
}

export default RunDetailsApplyStage;
