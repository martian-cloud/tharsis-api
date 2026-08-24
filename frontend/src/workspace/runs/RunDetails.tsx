import DoubleArrowIcon from '@mui/icons-material/DoubleArrow';
import { Alert, AlertTitle, IconButton, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import { useEffect, useMemo, useState } from 'react';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import { Navigate, Route, Routes, useParams } from 'react-router-dom';
import { useAgentCopilot } from '../../ai/AgentCopilotProvider';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import Link from '../../routes/Link';
import RunDetailsApplyStage from './RunDetailsApplyStage';
import RunDetailsPlanStage from './RunDetailsPlanStage';
import RunDetailsRunTaskStage from './taskstage/RunDetailsRunTaskStage';
import RunDetailsSidebar, { SidebarWidth } from './RunDetailsSidebar';
import { RunDetailsFragment_details$key } from './__generated__/RunDetailsFragment_details.graphql';
import { RunDetailsQuery } from './__generated__/RunDetailsQuery.graphql';
import { resolveCurrentStagePath } from './runStageNavigation';
import { usePageLayout } from '@/layout/PageLayoutContext';

// Each policy (task) stage has its own route, so the stage is a path segment rather than a query
// param and every stage of a run is a distinct URL.
const RUN_STAGE_NAMES = {
    pre_plan: 'Pre-Plan',
    plan: 'Plan',
    post_plan: 'Post-Plan',
    apply: 'Apply'
} as any;

// The run statuses that mean "ready, but waiting for the workspace slot" — one per workspace-gated
// phase. Mirrors models.QueuingRunStatuses on the server.
const QUEUING_STATUSES = new Set([
    'pre_plan_queuing', 'plan_queuing', 'pre_apply_queuing', 'apply_queuing',
]);

interface Props {
    fragmentRef: RunDetailsFragment_details$key
}

function RunDetails(props: Props) {
    const params = useParams();
    const runId = params.id as string;
    const stage = params['*'] || 'plan';
    const stageName = RUN_STAGE_NAMES[stage];

    usePageLayout('wide');
    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('md'));
    const { setState: setCopilotState } = useAgentCopilot();

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
                taskStages {
                    stageName
                    status
                    policyChecks {
                        status
                    }
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
                ...RunDetailsRunTaskStageFragment_taskStage
                ...RunDetailsApplyStageFragment_apply
            }
        }
    `, { id: runId }, { fetchPolicy: 'store-and-network' });

    useEffect(() => {
        setCopilotState({
            contextMessage: `The user is currently viewing run ID: ${runId} in workspace ${data.fullPath}.`,
            suggestions: [{ title: "Troubleshoot run", prompt: `Troubleshoot the run with ID: ${runId}` }]
        });
        return () => {
            setCopilotState(undefined);
        }
    }, [runId, data.fullPath, setCopilotState]);

    const onToggleSidebar = () => {
        setSidebarOpen(prev => !prev);
    };

    const onError = (error: MutationError) => {
        setError(error);
    };

    // The run is blocked by a user-held workspace lock: it is ready to proceed but cannot be admitted
    // until the workspace is unlocked. The *_QUEUING statuses are exactly the phases that wait on the
    // slot, so the run status alone says whether the lock is what is holding this run up.
    const displayLockWarning = useMemo(
        () => {
            const run = queryData.run;
            if (!run) {
                return false;
            }

            return run.workspace.locked && QUEUING_STATUSES.has(run.status)
        },
        [queryData.run?.status, queryData.run?.workspace?.locked]
    );

    // The stage the index route redirects to: the stage currently in progress, or the last completed
    // stage when nothing is in progress. Opening a run therefore lands on the stage the run is actually
    // sitting at instead of the furthest stage that merely exists.
    const defaultStagePath = useMemo(
        () => resolveCurrentStagePath(queryData.run),
        [queryData.run]
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
                            it can be unlocked within the <strong>State Settings</strong> section on the <Link to={`/groups/${queryData?.run?.workspace.fullPath}/-/settings`}>Settings</Link> page.
                        </Alert>}
                    {error && <Alert sx={{ marginBottom: 2 }} severity={error.severity}>
                        {error.message}
                    </Alert>}
                    {mobile && <Box display="flex" justifyContent="space-between">
                        <Typography variant="h6">{stageName} Details</Typography>
                        <IconButton onClick={onToggleSidebar}><DoubleArrowIcon sx={{ transform: 'rotate(180deg)' }} /></IconButton>
                    </Box>}
                    <Routes>
                        <Route index element={<Navigate to={defaultStagePath} replace />} />
                        {/* One route per policy stage, so which stage is being viewed is part of the
                            path. The stage each one renders is passed explicitly rather than read
                            back out of the URL. */}
                        <Route path="pre_plan" element={<RunDetailsRunTaskStage stageName="PRE_PLAN" fragmentRef={queryData.run} onError={onError} />} />
                        <Route path="plan" element={<RunDetailsPlanStage fragmentRef={queryData.run} onError={onError} />} />
                        <Route path="post_plan" element={<RunDetailsRunTaskStage stageName="POST_PLAN" fragmentRef={queryData.run} onError={onError} />} />
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
