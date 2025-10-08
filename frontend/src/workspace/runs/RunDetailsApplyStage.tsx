import { LoadingButton } from '@mui/lab';
import { Divider, Link as MuiLink, Paper, Tooltip, Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import { grey } from '@mui/material/colors';
import graphql from 'babel-plugin-relay/macro';
import humanizeDuration from 'humanize-duration';
import moment from 'moment';
import React, { useState } from 'react';
import Lottie from 'react-lottie-player';
import { useFragment, useMutation } from 'react-relay/hooks';
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
import { RunDetailsApplyStageApplyRunMutation } from './__generated__/RunDetailsApplyStageApplyRunMutation.graphql';
import { RunDetailsApplyStageFragment_apply$key } from './__generated__/RunDetailsApplyStageFragment_apply.graphql';
import NoRunnerAlert from './NoRunnerAlert';

interface Props {
    fragmentRef: RunDetailsApplyStageFragment_apply$key
    onError: (error: MutationError) => void
}

function RunDetailsApplyStage(props: Props) {
    const theme = useTheme();
    const [jobDialogOpen, setJobDialogOpen] = useState(false);

    const data = useFragment<RunDetailsApplyStageFragment_apply$key>(
        graphql`
        fragment RunDetailsApplyStageFragment_apply on Run
        {
            id
            status
            plan {
                summary {
                    resourceAdditions
                    resourceChanges
                    resourceDestructions
                }
                status
                ...RunDetailsPlanSummaryFragment_plan
            }
            apply {
                metadata {
                    createdAt
                    trn
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
                  ...JobLogsFragment_logs
                  ...RunJobDialog_currentJob
                }
            }
            ...RunVariablesFragment_variables
            ...ForceCancelRunAlertFragment_run
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

    const [tab, setTab] = useState('logs')

    const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
        setTab(newValue);
    };

    const timestamps = data.apply?.currentJob?.timestamps;
    const duration = timestamps?.finishedAt ?
        moment.duration(moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput))) : null;

    const applyStatusType = data.apply ? (RunStageStatusTypes[data.apply?.status] ?? { label: 'unknown', color: grey[500] }) : null;
    const StatusIcon = applyStatusType?.icon;

    return (
        <Box>
            {data.apply?.currentJob?.cancelRequested && data.apply?.status !== 'canceled' && <ForceCancelRunAlert fragmentRef={data} />}
            {data.apply?.currentJob && <NoRunnerAlert fragmentRef={data.apply.currentJob} sx={{ mb: 2 }} />}
            {data.apply && data.apply.status !== 'created' && data.apply.triggeredBy && <Box>
                {data.apply.status !== 'pending' && <Box
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
                        <Typography sx={{ paddingRight: '4px' }}>Apply triggered</Typography>
                        <Timestamp component="span" timestamp={data.apply.metadata.createdAt} />
                        <Typography sx={{ paddingLeft: '4px', paddingRight: '8px' }}>by</Typography>
                        <Gravatar width={20} height={20} email={data.apply.triggeredBy} />
                        <Typography
                            sx={{
                                paddingLeft: '8px',
                                [theme.breakpoints.down('lg')]: {
                                    display: 'none'
                                }
                            }}>
                            {data.apply.triggeredBy}
                        </Typography>
                    </Box>
                    <TRNButton trn={data.apply.metadata.trn} />
                </Box>}
                <Paper variant="outlined" sx={{ marginBottom: 2, p: 2 }} >
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
                                <Typography variant="h6">Apply {applyStatusType.label}</Typography>
                                {duration && <Typography variant="body2" color="textSecondary">
                                    Duration: {humanizeDuration(duration.asMilliseconds())}
                                </Typography>}
                            </Box>
                        </Box>
                    </Box>
                    {data.apply.status === 'errored' && !!data.apply.errorMessage && <React.Fragment>
                        <Divider sx={{ ml: -2, mr: -2, mt: 2 }} />
                        <RunDetailsErrorSummary errorMessage={data.apply.errorMessage} ml={-2} mr={-2} mb={-2} />
                    </React.Fragment>}
                    {data.apply.status === 'finished' && <React.Fragment>
                        <Divider sx={{ ml: -2, mr: -2, mt: 2 }} />
                        <RunDetailsPlanSummary fragmentRef={data.plan} ml={-2} mr={-2} completed={true} />
                        <Box mt={2}>
                            <Link color="secondary" to={'../plan?tab=changes'}>View changes</Link>
                        </Box>
                    </React.Fragment>}
                </Paper>
                {data.apply.status === 'pending' && <Box display="flex" justifyContent="center" marginTop={6}>
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
                        <Typography sx={{ marginBottom: 2 }} variant="h6" align="center">Apply operation is pending and will start shortly</Typography>
                        <Box display="flex" alignItems="center" marginLeft={2}>
                            <Typography sx={{ paddingRight: '4px' }} color="textSecondary">Triggered</Typography>
                            <Timestamp color="textSecondary" component="span" timestamp={data.apply.metadata.createdAt} />
                            <Typography sx={{ paddingLeft: '4px', paddingRight: '8px' }} color="textSecondary">by</Typography>
                            <Tooltip title={data.apply.triggeredBy}>
                                <Box>
                                    <Gravatar width={20} height={20} email={data.apply.triggeredBy} />
                                </Box>
                            </Tooltip>
                        </Box>
                    </Box>
                </Box>}
                {(data.apply && data.apply.currentJob && data.apply.status !== 'pending') && <Paper variant="outlined" sx={{ padding: 2, marginBottom: 2 }}>
                <Typography variant="body2" component="div">This apply has
                        <MuiLink
                            component="button"
                            color="secondary"
                            underline="hover"
                            onClick={() => setJobDialogOpen(true)}
                            variant="body2"
                            sx={{ marginLeft: '4px', fontWeight: 600 }}
                        >1 job
                    </MuiLink>
                </Typography>
            </Paper>}
                {data.apply.currentJob && data.apply.status !== 'pending' && <Box>
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs value={tab} onChange={onTabChange}>
                            <Tab label="Logs" value="logs" />
                            <Tab label="Variables" value="variables" />
                        </Tabs>
                    </Box>
                    {tab === 'logs' && <Box>
                        <JobLogs fragmentRef={data.apply.currentJob} enableAutoScrollByDefault={data.apply.status === 'running'} />
                    </Box>}
                    {tab === 'variables' && <Box marginTop={2}>
                        <RunVariables fragmentRef={data} />
                    </Box>}
                </Box>}
            </Box >}
            {(!data.apply || data.apply.status === 'created') && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    {['pending', 'plan_queued', 'planning'].includes(data.status) && <React.Fragment>
                        <Typography variant="h6">Apply has not started</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            The apply stage can be started after all previous stages have completed
                        </Typography>
                    </React.Fragment>}
                    {['canceled', 'errored'].includes(data.status) && <React.Fragment>
                        <Typography variant="h6">Apply not started</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            The apply stage was not started because the run has failed or was canceled
                        </Typography>
                    </React.Fragment>}
                    {data.status === 'planned_and_finished' && <React.Fragment>
                        <Typography variant="h6">Apply skipped</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            The apply stage has been skipped because the plan did not contain any changes
                        </Typography>
                    </React.Fragment>}
                    {data.status === 'planned' && <React.Fragment>
                        <Typography variant="h6">Apply has not started</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            All previous stages have completed so the apply is ready to be started
                        </Typography>
                        <LoadingButton loading={commitApplyRunInFlight} variant="outlined" color="primary" onClick={applyRun}>Start Apply</LoadingButton>
                    </React.Fragment>}
                </Box>
            </Box>}
            {jobDialogOpen && <RunJobDialog
                fragmentRef={data.apply?.currentJob as RunJobDialog_currentJob$key}
                onClose={() => setJobDialogOpen(false)}
            />}
        </Box>
    );
}

export default RunDetailsApplyStage;
