import React, { Suspense } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import { Box, Button, Chip, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, Link as MuiLink, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Tooltip, Typography } from "@mui/material";
import { useFragment, useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import IconButton from '@mui/material/IconButton';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import Link from '../../routes/Link';
import moment from 'moment';
import humanizeDuration from 'humanize-duration';
import JobStatusChip from './JobStatusChip';
import { RunJobDialog_jobs$key } from './__generated__/RunJobDialog_jobs.graphql';
import { RunJobDialogPlanQuery } from './__generated__/RunJobDialogPlanQuery.graphql';
import { RunJobDialogPlanFragment_jobs$key } from './__generated__/RunJobDialogPlanFragment_jobs.graphql';
import { RunJobDialogPlanPaginationQuery } from './__generated__/RunJobDialogPlanPaginationQuery.graphql';
import { RunJobDialogApplyQuery } from './__generated__/RunJobDialogApplyQuery.graphql';
import { RunJobDialogApplyFragment_jobs$key } from './__generated__/RunJobDialogApplyFragment_jobs.graphql';
import { RunJobDialogApplyPaginationQuery } from './__generated__/RunJobDialogApplyPaginationQuery.graphql';

const PAGE_SIZE = 20;

const planQuery = graphql`
    query RunJobDialogPlanQuery($id: String!, $first: Int!, $after: String) {
        node(id: $id) {
            ...on Run {
                id
                ...RunJobDialogPlanFragment_jobs
            }
        }
    }
`;

const applyQuery = graphql`
    query RunJobDialogApplyQuery($id: String!, $first: Int!, $after: String) {
        node(id: $id) {
            ...on Run {
                id
                ...RunJobDialogApplyFragment_jobs
            }
        }
    }
`;

interface Props {
    runId: string
    stage: 'plan' | 'apply'
    onSelectJob: (jobId: string) => void
    onClose: (confirm?: boolean) => void
}

function jobDuration(timestamps: { runningAt: unknown, finishedAt: unknown } | null | undefined) {
    return (timestamps?.finishedAt && timestamps?.runningAt)
        ? moment.duration(moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput)))
        : null;
}

function RunJobDialog(props: Props) {
    const { runId, stage, onSelectJob, onClose } = props;
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

    return (
        <Dialog
            open
            maxWidth="lg"
            fullWidth
            fullScreen={fullScreen}
        >
            <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                Jobs
                <IconButton
                    color="inherit"
                    size="small"
                    onClick={() => onClose()}
                >
                    <CloseIcon />
                </IconButton>
            </DialogTitle>
            <DialogContent dividers sx={{ flex: 1, padding: 2, minHeight: 600, display: 'flex', flexDirection: 'column' }}>
                <Suspense fallback={<Box
                    sx={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        width: '100%',
                        minHeight: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}>
                    <CircularProgress />
                </Box>}>
                    {stage === 'plan'
                        ? <PlanJobsLoader runId={runId} onSelectJob={onSelectJob} />
                        : <ApplyJobsLoader runId={runId} onSelectJob={onSelectJob} />}
                </Suspense>
            </DialogContent>
            {!fullScreen && <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Close
                </Button>
            </DialogActions>}
        </Dialog>
    );
}

interface LoaderProps {
    runId: string
    onSelectJob: (jobId: string) => void
}

function PlanJobsLoader({ runId, onSelectJob }: LoaderProps) {
    const queryData = useLazyLoadQuery<RunJobDialogPlanQuery>(planQuery, { id: runId, first: PAGE_SIZE }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, isLoadingNext } = usePaginationFragment<RunJobDialogPlanPaginationQuery, RunJobDialogPlanFragment_jobs$key>(
        graphql`
        fragment RunJobDialogPlanFragment_jobs on Run
        @refetchable(queryName: "RunJobDialogPlanPaginationQuery")
        {
            plan {
                jobs(first: $first, after: $after, sort: CREATED_AT_DESC) @connection(key: "RunJobDialogPlan_jobs") {
                    edges {
                        node {
                            id
                            ...RunJobDialog_jobs
                        }
                    }
                }
            }
        }
        `, queryData.node ?? null
    );

    const jobs = (data?.plan?.jobs.edges ?? []).flatMap((edge) => edge?.node ? [edge.node] : []);
    return <RunJobList fragmentRef={jobs} hasMore={hasNext} loadingMore={isLoadingNext} onLoadMore={() => loadNext(PAGE_SIZE)} onSelectJob={onSelectJob} />;
}

function ApplyJobsLoader({ runId, onSelectJob }: LoaderProps) {
    const queryData = useLazyLoadQuery<RunJobDialogApplyQuery>(applyQuery, { id: runId, first: PAGE_SIZE }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, isLoadingNext } = usePaginationFragment<RunJobDialogApplyPaginationQuery, RunJobDialogApplyFragment_jobs$key>(
        graphql`
        fragment RunJobDialogApplyFragment_jobs on Run
        @refetchable(queryName: "RunJobDialogApplyPaginationQuery")
        {
            apply {
                jobs(first: $first, after: $after, sort: CREATED_AT_DESC) @connection(key: "RunJobDialogApply_jobs") {
                    edges {
                        node {
                            id
                            ...RunJobDialog_jobs
                        }
                    }
                }
            }
        }
        `, queryData.node ?? null
    );

    const jobs = (data?.apply?.jobs.edges ?? []).flatMap((edge) => edge?.node ? [edge.node] : []);
    return <RunJobList fragmentRef={jobs} hasMore={hasNext} loadingMore={isLoadingNext} onLoadMore={() => loadNext(PAGE_SIZE)} onSelectJob={onSelectJob} />;
}

interface RunJobListProps {
    fragmentRef: RunJobDialog_jobs$key
    hasMore: boolean
    loadingMore: boolean
    onLoadMore: () => void
    onSelectJob: (jobId: string) => void
}

function RunJobList({ fragmentRef, hasMore, loadingMore, onLoadMore, onSelectJob }: RunJobListProps) {
    const theme = useTheme();

    const jobs = useFragment<RunJobDialog_jobs$key>(
        graphql`
        fragment RunJobDialog_jobs on Job @relay(plural: true)
        {
            id
            status
            tags
            runner {
                id
                name
                type
                groupPath
            }
            runnerPath
            metadata {
                createdAt
            }
            timestamps {
                pendingAt
                runningAt
                finishedAt
            }
        }
        `, fragmentRef
    );

    return (
        <TableContainer sx={{
            borderTop: `1px solid ${theme.palette.divider}`,
            borderLeft: `1px solid ${theme.palette.divider}`,
            borderRight: `1px solid ${theme.palette.divider}`,
            borderBottom: `1px solid ${theme.palette.divider}`,
            borderBottomLeftRadius: 4,
            borderBottomRightRadius: 4,
        }}>
            <Table
                sx={{ minWidth: 650, tableLayout: 'fixed' }}
                aria-label="run jobs"
            >
                <TableHead>
                    <TableRow>
                        <TableCell>Status</TableCell>
                        <TableCell>ID</TableCell>
                        <TableCell>Tags</TableCell>
                        <TableCell>Runner</TableCell>
                        <TableCell>Duration</TableCell>
                        <TableCell>Created</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {jobs.length === 0 && <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
                        <TableCell colSpan={6}>
                            <Typography variant="body2" color="textSecondary">No jobs</Typography>
                        </TableCell>
                    </TableRow>}
                    {jobs.map((job) => {
                        const duration = jobDuration(job.timestamps);
                        return (
                            <TableRow key={job.id} sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
                                <TableCell>
                                    <JobStatusChip status={job.status} onClick={() => onSelectJob(job.id)} />
                                </TableCell>
                                <TableCell>
                                    <MuiLink
                                        component="button"
                                        color="textPrimary"
                                        underline="hover"
                                        sx={{ cursor: 'pointer', fontWeight: 500 }}
                                        onClick={() => onSelectJob(job.id)}
                                    >{job.id.substring(0, 8)}...
                                    </MuiLink>
                                </TableCell>
                                <TableCell>
                                    {(job.tags && job.tags.length > 0) ? <Stack direction="row" spacing={1}>
                                        {job.tags.map((tag: any) => <Chip key={tag} size="small" color="secondary" label={tag} />)}
                                    </Stack> : <Typography variant="body2" color="textSecondary">None</Typography>}
                                </TableCell>
                                <TableCell>
                                    {job.runner ? <Link
                                        color="primary"
                                        sx={{ fontWeight: 500 }}
                                        to={job.runner.type === 'shared' ?
                                            `/admin/runners/${job.runner.id}`
                                            : `/groups/${job.runner.groupPath}/-/runners/${job.runner.id}`
                                        }
                                    >
                                        {job.runner.name}
                                    </Link> : <React.Fragment>--</React.Fragment>}
                                    {!job.runner && job.runnerPath && <React.Fragment>{job.runnerPath} (deleted)</React.Fragment>}
                                    {job.timestamps.pendingAt &&
                                        <Typography
                                            component="div"
                                            variant="caption"
                                        >claimed job {moment(job.timestamps.pendingAt as moment.MomentInput).fromNow()}
                                        </Typography>}
                                </TableCell>
                                <TableCell>
                                    {duration ? humanizeDuration(duration.asMilliseconds()) : '--'}
                                </TableCell>
                                <TableCell>
                                    <Tooltip title={job.metadata.createdAt}>
                                        <Box>{moment(job.metadata.createdAt as moment.MomentInput).fromNow()}</Box>
                                    </Tooltip>
                                </TableCell>
                            </TableRow>
                        );
                    })}
                    {hasMore && <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
                        <TableCell colSpan={6} align="center">
                            <Button
                                color="inherit"
                                disabled={loadingMore}
                                onClick={() => onLoadMore()}
                                startIcon={loadingMore ? <CircularProgress size={16} color="inherit" /> : undefined}
                            >
                                {loadingMore ? 'Loading...' : 'Load more'}
                            </Button>
                        </TableCell>
                    </TableRow>}
                </TableBody>
            </Table>
        </TableContainer>
    );
}

export default RunJobDialog;
