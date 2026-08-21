import { Box, CircularProgress, List, ListItemButton, Link as MuiLink, Popover, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import humanizeDuration from 'humanize-duration';
import moment from 'moment';
import { Suspense, useState } from 'react';
import { useFragment, useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import Timestamp from '../../../common/Timestamp';
import JobStatusChip from '../JobStatusChip';
import { RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs$key } from './__generated__/RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs.graphql';
import { RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery } from './__generated__/RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery.graphql';
import { RunTaskStagePolicyCheckPreviousJobsMenuQuery } from './__generated__/RunTaskStagePolicyCheckPreviousJobsMenuQuery.graphql';
import { RunTaskStagePolicyCheckPreviousJobsMenu_job$key } from './__generated__/RunTaskStagePolicyCheckPreviousJobsMenu_job.graphql';

// How many jobs a page of the popover loads.
const PAGE_SIZE = 10;

// The check is reached through runNode rather than the run's taskStages: a Relay pagination fragment
// cannot page a connection nested inside plural fields, and both taskStages and its policyChecks are
// lists. runNode addresses one node of the run directly, by the same path retryRunNode takes.
const query = graphql`
    query RunTaskStagePolicyCheckPreviousJobsMenuQuery($runId: String!, $nodePath: String!, $first: Int!, $after: String) {
        ...RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs
    }
`;

interface Props {
    runId: string
    // The check's node path within the run (e.g. "post_plan.opa"), which is how a single check is
    // addressed — a check is not a Node and has no query of its own.
    nodePath: string
    // The check's latest job, excluded from the list — it is what the log viewer shows by default.
    currentJobId?: string
    // Number of jobs before the current one, taken from the check's own totalCount so the link can
    // render (and count) without opening the popover and firing its query.
    previousJobCount: number
    // The job currently pinned in the URL, if any, so its row can be marked as selected.
    selectedJobId?: string | null
    onSelectJob: (jobId: string) => void
}

// RunTaskStagePolicyCheckPreviousJobsMenu is the "N previous jobs" link in the policy check panel's job logs
// header, and the popover it opens. Each retry of a check leaves its job behind; selecting one here
// points the log viewer at it.
function RunTaskStagePolicyCheckPreviousJobsMenu({ runId, nodePath, currentJobId, previousJobCount, selectedJobId, onSelectJob }: Props) {
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

    if (previousJobCount < 1) {
        return null;
    }

    return (
        <>
            <MuiLink
                component="button"
                underline="hover"
                variant="body2"
                sx={{ whiteSpace: 'nowrap', fontFamily: 'inherit' }}
                onClick={event => setAnchorEl(event.currentTarget)}
            >
                {previousJobCount} previous job{previousJobCount === 1 ? '' : 's'}
            </MuiLink>
            <Popover
                open={Boolean(anchorEl)}
                anchorEl={anchorEl}
                onClose={() => setAnchorEl(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
                {/* Mounted only while open so the jobs query isn't fired until the link is clicked. */}
                <Box sx={{ minWidth: 320 }}>
                    <Suspense fallback={
                        <Box sx={{ minHeight: 80, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                            <CircularProgress size={20} />
                        </Box>
                    }>
                        <PreviousJobsLoader
                            runId={runId}
                            nodePath={nodePath}
                            currentJobId={currentJobId}
                            selectedJobId={selectedJobId}
                            onSelectJob={jobId => {
                                setAnchorEl(null);
                                onSelectJob(jobId);
                            }}
                        />
                    </Suspense>
                </Box>
            </Popover>
        </>
    );
}

type LoaderProps = Omit<Props, 'previousJobCount'>;

function PreviousJobsLoader({ runId, nodePath, currentJobId, selectedJobId, onSelectJob }: LoaderProps) {
    const queryData = useLazyLoadQuery<RunTaskStagePolicyCheckPreviousJobsMenuQuery>(
        query,
        // One wider than a page: the connection leads with the current job, which the list drops,
        // so this still lands on a full page of previous jobs.
        { runId, nodePath, first: PAGE_SIZE + 1 },
        { fetchPolicy: 'store-and-network' },
    );

    const { data, loadNext, hasNext, isLoadingNext } = usePaginationFragment<RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery, RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs$key>(
        graphql`
        fragment RunTaskStagePolicyCheckPreviousJobsMenuFragment_jobs on Query
        @refetchable(queryName: "RunTaskStagePolicyCheckPreviousJobsMenuPaginationQuery")
        {
            runNode(runId: $runId, nodePath: $nodePath) {
                ...on PolicyCheck {
                    jobs(first: $first, after: $after, sort: CREATED_AT_DESC) @connection(key: "RunTaskStagePolicyCheckPreviousJobsMenu_jobs") {
                        edges {
                            node {
                                id
                                ...RunTaskStagePolicyCheckPreviousJobsMenu_job
                            }
                        }
                    }
                }
            }
        }
        `, queryData
    );

    const jobs = (data?.runNode?.jobs?.edges ?? []).flatMap(edge => edge?.node ? [edge.node] : []);
    // The current job leads the list (it is the newest) and is never one of the previous jobs the
    // popover offers, so it is dropped rather than counted against the page.
    const previous = jobs.filter(job => job.id !== currentJobId);

    return (
        <List dense disablePadding>
            {previous.length === 0 && !hasNext && (
                <Box sx={{ padding: 2 }}>
                    <Typography variant="body2" color="textSecondary">No previous jobs</Typography>
                </Box>
            )}
            {previous.map(job => (
                <PreviousJobItem
                    key={job.id}
                    fragmentRef={job}
                    selected={job.id === selectedJobId}
                    onSelect={() => onSelectJob(job.id)}
                />
            ))}
            {hasNext && (
                <ListItemButton
                    disabled={isLoadingNext}
                    onClick={() => loadNext(PAGE_SIZE)}
                    sx={{ justifyContent: 'center', gap: 1 }}
                >
                    {isLoadingNext && <CircularProgress size={14} color="inherit" />}
                    <Typography variant="body2" color={isLoadingNext ? 'textSecondary' : 'primary'}>
                        {isLoadingNext ? 'Loading...' : 'Show more'}
                    </Typography>
                </ListItemButton>
            )}
        </List>
    );
}

interface ItemProps {
    fragmentRef: RunTaskStagePolicyCheckPreviousJobsMenu_job$key
    selected: boolean
    onSelect: () => void
}

function PreviousJobItem({ fragmentRef, selected, onSelect }: ItemProps) {
    const job = useFragment<RunTaskStagePolicyCheckPreviousJobsMenu_job$key>(
        graphql`
        fragment RunTaskStagePolicyCheckPreviousJobsMenu_job on Job
        {
            id
            status
            metadata {
                createdAt
            }
            timestamps {
                runningAt
                finishedAt
            }
        }
        `, fragmentRef
    );

    const durationMs = job.timestamps.runningAt && job.timestamps.finishedAt
        ? moment(job.timestamps.finishedAt as moment.MomentInput).diff(moment(job.timestamps.runningAt as moment.MomentInput))
        : null;

    return (
        <ListItemButton
            selected={selected}
            onClick={onSelect}
            sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}
        >
            <JobStatusChip status={job.status} />
            <Box sx={{ minWidth: 0, flex: 1 }}>
                <Typography variant="body2" sx={{ fontWeight: 500 }}>
                    {job.id.substring(0, 8)}...
                </Typography>
                <Typography variant="caption" color="textSecondary" component="div">
                    <Timestamp timestamp={job.metadata.createdAt as string} variant="caption" />
                    {durationMs !== null && ` · took ${humanizeDuration(durationMs)}`}
                </Typography>
            </Box>
        </ListItemButton>
    );
}

export default RunTaskStagePolicyCheckPreviousJobsMenu;
