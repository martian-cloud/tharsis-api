import DoubleArrowIcon from '@mui/icons-material/DoubleArrow';
import { Alert, Box, CircularProgress, IconButton, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Suspense, useState } from 'react';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import Gravatar from '../../../common/Gravatar';
import { JobIcon } from '../../../common/Icons';
import NamespaceBreadcrumbs from '../../../namespace/NamespaceBreadcrumbs';
import Timestamp from '../../../common/Timestamp';
import JobDetailsLogsCard from './JobDetailsLogsCard';
import JobDetailsResourceUsageCard from './JobDetailsResourceUsageCard';
import JobDetailsSidebar, { JobSidebarWidth } from './JobDetailsSidebar';
import OutdatedProtocolAlert from '../OutdatedProtocolAlert';
import { useJobStatusColor } from './jobUtils';
import { JobDetailsQuery } from './__generated__/JobDetailsQuery.graphql';
import { alpha, useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { usePageLayout } from '@/layout/PageLayoutContext';

const jobDetailsQuery = graphql`
    query JobDetailsQuery($id: String!) {
        node(id: $id) {
            ...on Job {
                id
                status
                type
                tags
                forceCanceled
                ...OutdatedProtocolAlertFragment_job
                runner {
                    id
                    name
                    type
                    groupPath
                }
                runnerPath
                workspace {
                    fullPath
                }
                metadata {
                    createdAt
                    trn
                }
                run {
                    id
                    createdBy
                    status
                    hasAdvisoryFailures
                    workspace {
                        fullPath
                    }
                }
                timestamps {
                    pendingAt
                    runningAt
                    finishedAt
                }
                ...JobDetailsResourceUsageCardFragment_job
            }
        }
    }
`;

function JobDetails() {
    const { jobId } = useParams<{ jobId: string }>();

    // The job page is its own top-level route now, so it sets the wide layout itself.
    usePageLayout('wide');

    return (
        <Suspense fallback={
            <Box display="flex" justifyContent="center" mt={6}>
                <CircularProgress />
            </Box>
        }>
            <JobDetailsContent jobId={jobId!} />
        </Suspense>
    );
}

function JobDetailsContent({ jobId }: { jobId: string }) {
    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('md'));
    const [sidebarOpen, setSidebarOpen] = useState(false);

    const data = useLazyLoadQuery<JobDetailsQuery>(
        jobDetailsQuery,
        { id: jobId },
        { fetchPolicy: 'store-and-network' }
    );

    const job = data.node;
    const status = job?.status ?? '';
    const statusColor = useJobStatusColor(status);

    if (!job?.id || !job.metadata || !job.timestamps || !job.status) {
        return <Alert severity="error">Job not found</Alert>;
    }

    const workspacePath = job.workspace?.fullPath ?? '';
    const onToggleSidebar = () => setSidebarOpen(prev => !prev);
    const createdBy = job.run?.createdBy;

    return (
        <Box>
            <JobDetailsSidebar
                job={job}
                status={status}
                open={sidebarOpen}
                temporary={mobile}
                onClose={onToggleSidebar}
            />

            <Box sx={{ paddingRight: !mobile ? `${JobSidebarWidth}px` : 0 }}>
                <NamespaceBreadcrumbs
                    namespacePath={workspacePath}
                    childRoutes={[
                        { title: 'jobs', path: 'jobs', disabled: true },
                        { title: `${jobId.substring(0, 8)}...`, path: `/groups/${workspacePath}/-/jobs/${jobId}` },
                    ]}
                />

                <OutdatedProtocolAlert fragmentRef={job} />

                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, pt: 1, mb: 3 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.75, minWidth: 0 }}>
                        <Box sx={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            width: 44,
                            height: 44,
                            borderRadius: 2,
                            flexShrink: 0,
                            color: statusColor,
                            backgroundColor: alpha(statusColor, 0.12),
                            border: `1px solid ${alpha(statusColor, 0.25)}`,
                        }}>
                            <JobIcon />
                        </Box>
                        <Box sx={{ minWidth: 0 }}>
                            <Typography variant="h6" component="h1" sx={{ fontWeight: 600, lineHeight: 1.2, fontFamily: 'monospace' }}>
                                {`${jobId.substring(0, 8)}…`}
                            </Typography>
                            <Typography variant="body2" component="div" color="textSecondary">
                                Triggered{' '}
                                <Timestamp variant="body2" component="span" color="textSecondary" timestamp={job.metadata.createdAt} />
                                {createdBy && <>
                                    {' '}by{' '}
                                    <Gravatar sx={{ display: 'inline-block', verticalAlign: 'middle', mr: 0.5 }} width={18} height={18} email={createdBy} />
                                    {createdBy}
                                </>}
                            </Typography>
                        </Box>
                    </Box>
                    {mobile && (
                        <IconButton aria-label="Show job details" onClick={onToggleSidebar} sx={{ flexShrink: 0 }}>
                            <DoubleArrowIcon sx={{ transform: 'rotate(180deg)' }} />
                        </IconButton>
                    )}
                </Box>

                <JobDetailsResourceUsageCard fragmentRef={job} />

                <JobDetailsLogsCard jobId={jobId} />
            </Box>
        </Box>
    );
}

export default JobDetails;
