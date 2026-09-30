import React, { Suspense } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import { Box, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, IconButton } from '@mui/material';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { useFragment, useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import { useNavigate } from 'react-router-dom';
import graphql from 'babel-plugin-relay/macro';
import RunJobList from './RunJobList';
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
    workspacePath: string
    onClose: () => void
}

function RunJobDialog({ runId, stage, workspacePath, onClose }: Props) {
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));
    const navigate = useNavigate();

    const handleSelectJob = (jobId: string) => {
        navigate(`/groups/${workspacePath}/-/jobs/${jobId}`);
        onClose();
    };

    return (
        <Dialog open maxWidth="md" fullWidth fullScreen={fullScreen}>
            <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', py: 1.5 }}>
                Jobs
                <IconButton color="inherit" size="small" onClick={onClose} aria-label="close">
                    <CloseIcon />
                </IconButton>
            </DialogTitle>
            <DialogContent dividers sx={{ p: 0, minHeight: 400 }}>
                <Suspense fallback={
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 200 }}>
                        <CircularProgress />
                    </Box>
                }>
                    {stage === 'plan'
                        ? <PlanJobsLoader runId={runId} onSelectJob={handleSelectJob} />
                        : <ApplyJobsLoader runId={runId} onSelectJob={handleSelectJob} />
                    }
                </Suspense>
            </DialogContent>
            {!fullScreen && (
                <DialogActions>
                    <Button color="inherit" onClick={onClose}>Close</Button>
                </DialogActions>
            )}
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

    const jobs = (data?.plan?.jobs.edges ?? []).flatMap(edge => edge?.node ? [edge.node] : []);
    return <JobList fragmentRefs={jobs} hasMore={hasNext} loadingMore={isLoadingNext} onLoadMore={() => loadNext(PAGE_SIZE)} onSelectJob={onSelectJob} />;
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

    const jobs = (data?.apply?.jobs.edges ?? []).flatMap(edge => edge?.node ? [edge.node] : []);
    return <JobList fragmentRefs={jobs} hasMore={hasNext} loadingMore={isLoadingNext} onLoadMore={() => loadNext(PAGE_SIZE)} onSelectJob={onSelectJob} />;
}

interface JobListProps {
    fragmentRefs: RunJobDialog_jobs$key
    hasMore: boolean
    loadingMore: boolean
    onLoadMore: () => void
    onSelectJob: (jobId: string) => void
}

function JobList({ fragmentRefs, hasMore, loadingMore, onLoadMore, onSelectJob }: JobListProps) {
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
        `, fragmentRefs
    );

    return (
        <RunJobList
            jobs={jobs}
            selectedJobId={null}
            hasMore={hasMore}
            loadingMore={loadingMore}
            onSelectJob={onSelectJob}
            onLoadMore={onLoadMore}
        />
    );
}

export default RunJobDialog;
