import DoubleArrowIcon from '@mui/icons-material/DoubleArrow';
import { Alert, AlertTitle, IconButton, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import { useMemo, useState } from 'react';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import { Route, Routes, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import RunDetailsApplyStage from './RunDetailsApplyStage';
import RunDetailsPlanStage from './RunDetailsPlanStage';
import RunDetailsSidebar, { SidebarWidth } from './RunDetailsSidebar';
import { RunDetailsFragment_details$key } from './__generated__/RunDetailsFragment_details.graphql';
import { RunDetailsQuery } from './__generated__/RunDetailsQuery.graphql';
import Link from '../../routes/Link';
import moment from 'moment';

const RUN_STAGE_NAMES = {
    plan: 'Plan',
    apply: 'Apply'
} as any;

interface Props {
    fragmentRef: RunDetailsFragment_details$key
}

function RunDetails(props: Props) {
    const params = useParams();
    const runId = params.id as string;
    const stage = params['*'] || 'plan';

    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('md'));

    const [sidebarOpen, setSidebarOpen] = useState(false);
    const [error, setError] = useState<MutationError>();

    const data = useFragment<RunDetailsFragment_details$key>(
        graphql`
        fragment RunDetailsFragment_details on Workspace
        {
            id
            fullPath
        }
      `, props.fragmentRef);

    const queryData = useLazyLoadQuery<RunDetailsQuery>(graphql`
        query RunDetailsQuery($id: String!) {
            run(id: $id) {
                status
                plan {
                    status
                }
                apply {
                    status
                }
                workspace {
                    fullPath
                    locked
                    metadata {
                        updatedAt
                    }
                }
                ...RunDetailsSidebarFragment_details
                ...RunDetailsPlanStageFragment_plan
                ...RunDetailsApplyStageFragment_apply
            }
        }
    `, { id: runId }, { fetchPolicy: 'store-and-network' });

    const onToggleSidebar = () => {
        setSidebarOpen(prev => !prev);
    };

    const onError = (error: MutationError) => {
        setError(error);
    };

    const displayLockWarning = useMemo(
        () => {
            const run = queryData.run;
            if (!run) {
                return false;
            }
            if ((run.plan.status == 'queued' || run.apply?.status == 'queued') && run.workspace.locked) {
                const duration = moment.duration(moment().diff(moment(run.workspace.metadata.updatedAt as moment.MomentInput)));
                // Only display the warning if the workspace has been locked for at least 10 seconds
                return duration.asSeconds() > 10;
            }
            return false;
        },
        [queryData.run?.status, queryData.run?.workspace?.locked, queryData.run?.workspace?.metadata?.updatedAt]
    );

    return queryData.run ? (
        <Box>
            <RunDetailsSidebar
                fragmentRef={queryData.run}
                stage={stage}
                open={sidebarOpen}
                temporary={mobile}
                onClose={onToggleSidebar}
                onError={onError}
            />
            <Box>
                <Box paddingRight={!mobile ? `${SidebarWidth}px` : 0}>
                    <NamespaceBreadcrumbs
                        namespacePath={data.fullPath}
                        childRoutes={[
                            { title: "runs", path: 'runs' },
                            { title: `${runId.substring(0, 8)}...`, path: runId }
                        ]}
                    />
                    {displayLockWarning &&
                        <Alert severity="warning" variant="outlined" sx={{ marginBottom: 2 }}>
                            <AlertTitle>Workspace is currently locked</AlertTitle>
                            A lock prevents new runs from starting. If the workspace was manually locked,
                            it can be unlocked within the <strong>State Settings</strong> section on the
                            <Link to={`/groups/${queryData?.run?.workspace.fullPath}/-/settings`}>Settings</Link> page.
                        </Alert>}
                    {error && <Alert sx={{ marginBottom: 2 }} severity={error.severity}>
                        {error.message}
                    </Alert>}
                    {mobile && <Box display="flex" justifyContent="space-between">
                        <Typography variant="h6">{RUN_STAGE_NAMES[stage]} Details</Typography>
                        <IconButton onClick={onToggleSidebar}><DoubleArrowIcon sx={{ transform: 'rotate(180deg)' }} /></IconButton>
                    </Box>}
                    <Routes>
                        <Route index element={<RunDetailsPlanStage fragmentRef={queryData.run} onError={onError} />} />
                        <Route path="plan" element={<RunDetailsPlanStage fragmentRef={queryData.run} onError={onError} />} />
                        <Route path="apply" element={<RunDetailsApplyStage fragmentRef={queryData.run} onError={onError} />} />
                    </Routes>
                </Box>
            </Box>
        </Box>
    ) : <Box>
        <Typography mt={4} variant="h6" color="textSecondary" align="center">Run not found</Typography>
    </Box>;
}

export default RunDetails;
