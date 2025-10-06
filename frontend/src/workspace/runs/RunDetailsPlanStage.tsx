import { useState } from 'react';
import LoadingButton from '@mui/lab/LoadingButton';
import { CircularProgress, Divider, Link as MuiLink, Paper, Tooltip, Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { grey } from '@mui/material/colors';
import graphql from 'babel-plugin-relay/macro';
import humanizeDuration from 'humanize-duration';
import moment from 'moment';
import React, { Suspense, useMemo } from 'react';
import Lottie from 'react-lottie-player';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import { MutationError } from '../../common/error';
import RocketLottieFileJson from '../../lotties/rocket-in-space-lottie.json';
import Link from '../../routes/Link';
import ForceCancelRunAlert from './ForceCancelRunAlert';
import JobLogs from './JobLogs';
import RunDetailsErrorSummary from './RunDetailsErrorSummary';
import RunDetailsPlanSummary from './RunDetailsPlanSummary';
import RunStageStatusTypes from './RunStageStatusTypes';
import RunVariables from './RunVariables';
import RunJobDialog from './RunJobDialog';
import { RunJobDialog_currentJob$key } from './__generated__/RunJobDialog_currentJob.graphql';
import { RunDetailsPlanStageApplyRunMutation } from './__generated__/RunDetailsPlanStageApplyRunMutation.graphql';
import { RunDetailsPlanStageFragment_plan$key } from './__generated__/RunDetailsPlanStageFragment_plan.graphql';
import RunDetailsPlanDiffViewer, { MaxDiffSize } from './plandiff/RunDetailsPlanDiffViewer';
import NoRunnerAlert from './NoRunnerAlert';

interface Props {
    fragmentRef: RunDetailsPlanStageFragment_plan$key
    onError: (error: MutationError) => void
}

function RunDetailsPlanStage(props: Props) {
    const theme = useTheme();
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const tab = searchParams.get('tab') ?? 'logs';
    const [jobDialogOpen, setJobDialogOpen] = useState(false);

    const data = useFragment<RunDetailsPlanStageFragment_plan$key>(
        graphql`
        fragment RunDetailsPlanStageFragment_plan on Run
        {
            id
            createdBy
            workspace {
                locked
                metadata {
                    updatedAt
                }
            }
            plan {
                metadata {
                    createdAt
                    trn
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
                  ...JobLogsFragment_logs
                  ...RunJobDialog_currentJob
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

    const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
        navigate({
            search: `?tab=${newValue}`
        }, {
            replace: true
        });
    };

    const duration = useMemo(() => {
        const timestamps = data.plan.currentJob?.timestamps;
        return timestamps?.finishedAt ?
            moment.duration(moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput))) : null;
    }, [data.plan.currentJob]);

    const planStatusType = RunStageStatusTypes[data.plan.status] ?? { label: 'unknown', color: grey[500] };
    const StatusIcon = planStatusType.icon;

    const maxDiffSizeExceeded = useMemo(() => data.plan.diffSize > MaxDiffSize, [data.plan.diffSize]);

    return (
        <Box>
            {data.plan.currentJob?.cancelRequested && data.plan.status !== 'canceled' && <ForceCancelRunAlert fragmentRef={data} />}
            {data.plan.currentJob && <NoRunnerAlert fragmentRef={data.plan.currentJob} sx={{ mb: 2 }} />}
            {data.plan.status !== 'pending' && <Box
                sx={{
                    paddingTop: 1,
                    marginBottom: 2,
                    display: 'flex',
                    flexDirection: { xs: 'column', md: 'row' },
                    alignItems: { xs: 'flex-start', md: 'center' },
                    justifyContent: { xs: 'flex-start', md: 'space-between' },
                    gap: { xs: 1 },
                }}>
                <Box display="flex" alignItems="center">
                    <Typography sx={{ paddingRight: '4px' }}>Plan triggered</Typography>
                    <Timestamp component="span" timestamp={data.plan.metadata.createdAt} />
                    <Typography sx={{ paddingLeft: '4px', paddingRight: '8px' }}>by</Typography>
                    <Gravatar width={20} height={20} email={data.createdBy} />
                    <Typography
                        sx={{
                            paddingLeft: '8px',
                            [theme.breakpoints.down('lg')]: {
                                display: 'none'
                            }
                        }}>
                        {data.createdBy}
                    </Typography>
                </Box>
                <TRNButton trn={data.plan.metadata.trn} />
            </Box>}
            {data.plan.status !== 'pending' && <Paper variant="outlined" sx={{ marginBottom: 2, p: 2 }} >
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    [theme.breakpoints.down('md')]: {
                        flexDirection: 'column',
                        alignItems: 'flex-start',
                        '& > *:not(:last-child)': {
                            marginBottom: 2
                        },
                    }
                }}>
                    <Box display="flex" alignItems="center">
                        <StatusIcon sx={{ width: 32, height: 32, mr: 2 }} />
                        <Box>
                            <Typography variant="h6">Plan {planStatusType.label}</Typography>
                            {duration && <Typography variant="body2" color="textSecondary">Duration: {humanizeDuration(duration.asMilliseconds())}</Typography>}
                        </Box>
                    </Box>
                    {data.apply && data.apply.status === 'created' && data.plan.hasChanges && <Box>
                        <LoadingButton loading={commitApplyRunInFlight} variant="outlined" size="medium" onClick={applyRun}>
                            Start Apply
                        </LoadingButton>
                    </Box>}
                </Box>
                {data.plan.status === 'errored' && !!data.plan.errorMessage && <React.Fragment>
                    <Divider sx={{ ml: -2, mr: -2, mt: 2 }} />
                    <RunDetailsErrorSummary errorMessage={data.plan.errorMessage} ml={-2} mr={-2} mb={-2} />
                </React.Fragment>}
                {data.plan.status === 'finished' && <React.Fragment>
                    <Divider sx={{ ml: -2, mr: -2, mt: 2 }} />
                    {!data.plan.hasChanges && <Typography mt={2} variant="body2">
                        This plan does not contain any changes to apply
                    </Typography>}
                    {data.plan.hasChanges && <React.Fragment>
                        <RunDetailsPlanSummary fragmentRef={data.plan} ml={-2} mr={-2} completed={false} />
                        <Box mt={2}>
                            <Link color="secondary" to={'?tab=changes'}>View changes</Link>
                        </Box>
                    </React.Fragment>}
                </React.Fragment>}
            </Paper>}
            {data.plan.status === 'pending' && <Box display="flex" justifyContent="center" marginTop={6}>
                <Box display="flex" flexDirection="column" alignItems="center">
                    <Lottie
                        renderer={undefined}
                        rendererSettings={undefined}
                        audioFactory={undefined}
                        animationData={RocketLottieFileJson}
                        loop={true}
                        play
                        style={{ width: 250, height: 250 }}
                    />
                    <Typography sx={{ marginBottom: 2 }} variant="h6" align="center">Plan operation is pending and will start shortly</Typography>
                    <Box display="flex" alignItems="center" marginLeft={2}>
                        <Typography sx={{ paddingRight: '4px' }} color="textSecondary">Triggered</Typography>
                        <Timestamp color="textSecondary" component="span" timestamp={data.plan.metadata.createdAt} />
                        <Typography sx={{ paddingLeft: '4px', paddingRight: '8px' }} color="textSecondary">by</Typography>
                        <Tooltip title={data.createdBy}>
                            <Box>
                                <Gravatar width={20} height={20} email={data.createdBy} />
                            </Box>
                        </Tooltip>
                    </Box>
                </Box>
            </Box>}
            {data.plan.currentJob && data.plan.status !== 'pending' && <Paper variant="outlined" sx={{ padding: 2, marginBottom: 2 }}>
                <Typography variant="body2" component="div">This plan has
                    <MuiLink
                        component="button"
                        color="secondary"
                        underline="hover"
                        onClick={() => setJobDialogOpen(true)}
                        variant="body2"
                        sx={{ marginLeft: '4px', fontWeight: 600 }}
                    >1 Job
                    </MuiLink>
                </Typography>
            </Paper>}
            {data.plan.currentJob && data.plan.status !== 'pending' && <Box>
                <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                    <Tabs value={tab} onChange={onTabChange}>
                        <Tab label="Logs" value="logs" />
                        <Tab label="Variables" value="variables" />
                        <Tab label="Changes" value="changes" />
                    </Tabs>
                </Box>
                {tab === 'logs' && <Box>
                    <JobLogs fragmentRef={data.plan.currentJob} />
                    {data.plan.hasChanges && data.plan.status === 'finished' && <Box mt={2}>
                        <Link color="secondary" to={'?tab=changes'}>View plan changes</Link>
                    </Box>}
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
                    {data.plan.status !== 'finished' && <Paper
                        variant="outlined"
                        sx={{
                            minHeight: 100,
                            display: 'flex',
                            flexDirection: 'column',
                            justifyContent: 'center',
                        }}
                    >
                        <Typography color="textSecondary" align="center">
                            {['pending', 'queued', 'running'].includes(data.plan.status) ?
                                'Changes will be displayed once the plan has completed' : 'This plan does not contain any changes'}
                        </Typography>
                    </Paper>}
                </Box>}
            </Box>}
            {jobDialogOpen && <RunJobDialog
                fragmentRef={data.plan?.currentJob as RunJobDialog_currentJob$key}
                onClose={() => setJobDialogOpen(false)}
            />}
        </Box>
    );
}

export default RunDetailsPlanStage;
