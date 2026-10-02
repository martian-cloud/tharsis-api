import NoResults from '@/common/NoResults';
import { Alert, AlertTitle, Avatar, Box, Button, Chip, FormControlLabel, Paper, Stack, Switch, Tab, Tabs, Tooltip, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React, { useEffect, useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom';
import ConfirmationDialog from '../common/ConfirmationDialog';
import TabContent from '../common/TabContent';
import TRNButton from '../common/TRNButton';
import { MutationError } from '../common/error';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import WorkspaceDetailsStatusPanel from './WorkspaceDetailsStatusPanel';
import WorkspaceDetailsEmpty from './WorkspaceDetailsEmpty';
import { WorkspaceDetailsIndexFragment_workspace$key } from './__generated__/WorkspaceDetailsIndexFragment_workspace.graphql';
import { WorkspaceDetailsIndex_DestroyWorkspaceMutation } from './__generated__/WorkspaceDetailsIndex_DestroyWorkspaceMutation.graphql';
import { WorkspaceDetailsIndex_ReconcileWorkspaceMutation } from './__generated__/WorkspaceDetailsIndex_ReconcileWorkspaceMutation.graphql';
import LabelList from './labels/LabelList';
import StateVersionDependencies from './state/StateVersionDependencies';
import StateVersionFile from './state/StateVersionFile';
import StateVersionInputVariables from './state/StateVersionInputVariables';
import StateVersionOutputs from './state/StateVersionOutputs';
import StateVersionResources from './state/StateVersionResources';
import StateVersionCheckResults from './state/StateVersionCheckResults';
import WorkspaceDetailsDriftDetection from './WorkspaceDetailsDriftDetection';
import WorkspaceNotificationPreference from '../notifications/WorkspaceNotificationPreference';
import NamespaceFavoriteButton from '../common/NamespaceFavoriteButton';
import { useAgentCopilot } from '../ai/AgentCopilotProvider';

const DRIFT_ALERT_DESCRIPTION = "This workspace has drifted from its configuration; this can happen if the resources were modified outside of Tharsis, or if the infrastructure was changed directly through the cloud provider console."

interface Props {
    fragmentRef: WorkspaceDetailsIndexFragment_workspace$key
}

function WorkspaceDetailsIndex(props: Props) {
    const { fragmentRef } = props;
    const theme = useTheme();
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const [showDestroyRunConfirmationDialog, setShowDestroyRunConfirmationDialog] = useState<boolean>(false);
    const [destroyAutoApply, setDestroyAutoApply] = useState<boolean>(false);
    const [showReconcileDialog, setShowReconcileDialog] = useState(false);
    const [error, setError] = useState<MutationError>();
    const { setState: setCopilotState } = useAgentCopilot();

    const tab = searchParams.get('tab') ?? 'resources';

    const data = useFragment<WorkspaceDetailsIndexFragment_workspace$key>(
        graphql`
      fragment WorkspaceDetailsIndexFragment_workspace on Workspace
      {
        id
        name
        description
        fullPath
        locked
        destroyed
        preventDestroyPlan
        labels {
            key
            value
        }
        metadata {
            trn
        }
        assessment {
            hasDrift
        }
        ...WorkspaceDetailsEmptyFragment_workspace
        ...WorkspaceDetailsStatusPanelFragment_workspace
        ...WorkspaceNotificationPreferenceFragment_workspace
        currentApplyRun {
            id
        }
        currentStateVersion {
            id
            ...StateVersionOutputsFragment_outputs
            inventory {
                ...StateVersionResourcesFragment_resources
                ...StateVersionDependenciesFragment_dependencies
                ...StateVersionCheckResultsFragment_checkResults
            }
            ...StateVersionFileFragment_stateVersion
            run {
                id
                ...StateVersionInputVariablesFragment_variables
            }
        }
        ...WorkspaceDetailsDriftDetectionFragment_workspace
      }
    `, fragmentRef);

    const workspaceDestroyed = data.destroyed;

    const [commitDestroyWorkspace, destroyWorkspaceIsInFlight] = useMutation<WorkspaceDetailsIndex_DestroyWorkspaceMutation>(graphql`
        mutation WorkspaceDetailsIndex_DestroyWorkspaceMutation($input: DestroyWorkspaceInput!) {
            destroyWorkspace (input: $input) {
                run {
                    id
                }
                problems {
                    message
                    field
                    type
                }
            }
        }`)

    const onDestroyRun = () => {
        commitDestroyWorkspace({
            variables: {
                input: {
                    workspacePath: data.fullPath,
                    autoApply: destroyAutoApply,
                }
            },
            onCompleted: data => {
                if (data.destroyWorkspace.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.destroyWorkspace.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.destroyWorkspace) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    setShowDestroyRunConfirmationDialog(false)
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        })
    };

    const onDestroyConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            onDestroyRun();
        }
        setShowDestroyRunConfirmationDialog(false);
        setDestroyAutoApply(false);
    };

    const [commitReconcile, isReconcileInFlight] = useMutation<WorkspaceDetailsIndex_ReconcileWorkspaceMutation>(graphql`
        mutation WorkspaceDetailsIndex_ReconcileWorkspaceMutation($input: ReconcileWorkspaceInput!) {
            reconcileWorkspace(input: $input) {
                run {
                    id
                }
                problems {
                    message
                    field
                    type
                }
            }
        }`)

    const onReconcileDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commitReconcile({
                variables: {
                    input: {
                        workspaceId: data.id
                    }
                },
                onCompleted: data => {
                    if (data.reconcileWorkspace.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.reconcileWorkspace.problems.map(problem => problem.message).join('; ')
                        })
                    } else if (!data.reconcileWorkspace.run) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        })
                    } else {
                        setShowReconcileDialog(false);
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: error.message
                    })
                }
            })
        }
        setShowReconcileDialog(false);
    };

    const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
        navigate({
            search: `?tab=${newValue}`
        }, {
            replace: true
        });
    };

    useEffect(() => {
        setCopilotState({
            contextMessage: `The user is currently viewing workspace ${data.fullPath} with ID: ${data.id}.`,
            suggestions: []
        });
        return () => {
            setCopilotState(undefined);
        }
    }, [data.fullPath, setCopilotState]);

    return (
        <Box>
            <NamespaceBreadcrumbs namespacePath={data.fullPath} />
            <Box
                sx={{
                    display: "flex",
                    justifyContent: "space-between",
                    mb: 4,
                    [theme.breakpoints.down('lg')]: {
                        flexDirection: 'column',
                        alignItems: 'flex-start',
                        '& > *:not(:last-child)': { mb: 2 },
                    }
                }}>
                <Box display="flex" alignItems="center">
                    <Avatar sx={{ width: 56, height: 56, marginRight: 2, bgcolor: 'avatar.default' }} variant="rounded">{data.name[0].toUpperCase()}</Avatar>
                    <Stack>
                        <Stack direction="row" spacing={1} alignItems="center">
                            <Typography variant="h5" sx={{ fontWeight: "bold" }}>{data.name}</Typography>
                            {workspaceDestroyed && <Chip size="small" label="Destroyed" sx={{ color: 'runStatus.destroy' }} />}
                        </Stack>
                        <Typography color="textSecondary" variant="subtitle2">{data.description}</Typography>
                        {data.labels && data.labels.length > 0 && (
                            <Box sx={{ mt: 1, display: "flex", justifyContent: "flex-end" }}>
                                <LabelList
                                    labels={[...data.labels]}
                                    size="small"
                                    maxVisible={6}
                                />
                            </Box>
                        )}
                    </Stack>
                </Box>
                <Box sx={{ display: 'flex', gap: 1, flexDirection: { xs: 'column', md: 'row' }, alignItems: { xs: 'stretch', md: 'center' } }}>
                    <Stack direction="row" spacing={1} alignItems="center">
                        <NamespaceFavoriteButton
                            namespacePath={data.fullPath}
                            namespaceType="WORKSPACE"
                        />
                        <WorkspaceNotificationPreference fragmentRef={data} />
                        <TRNButton trn={data.metadata.trn} size="small" />
                    </Stack>
                    {(!workspaceDestroyed && data.currentStateVersion && data.currentStateVersion.run) && (
                        <Tooltip
                            title={data.preventDestroyPlan ? "Prevent Destroy Run is enabled for this workspace." : "Create a destroy run, which destroys all resources in this workspace."}
                            placement="top"
                        >
                            <Box component="span" sx={{ width: { xs: '100%', md: 'auto' } }}>
                                <Button
                                    fullWidth
                                    size="small"
                                    variant="outlined"
                                    color="error"
                                    disabled={data.preventDestroyPlan}
                                    onClick={() => setShowDestroyRunConfirmationDialog(true)}
                                >
                                    Destroy Workspace
                                </Button>
                            </Box>
                        </Tooltip>
                    )}
                </Box>
            </Box>

            {data.locked &&
                <Alert
                    sx={{ mb: 2 }}
                    variant="outlined"
                    severity='warning'>
                    <AlertTitle>Workspace locked</AlertTitle>
                    This workspace is locked. New runs are prevented from starting and the state version cannot be modified. A lock is often used while manually updating the state version.
                    <Box sx={{ mt: 1, display: "flex", justifyContent: "flex-start" }}>
                        <Button
                            variant="outlined"
                            color="warning"
                            size="small"
                            component={RouterLink}
                            to={`/groups/${data.fullPath}/-/settings?section=state`}
                        >
                            Manage Workspace Lock
                        </Button>
                    </Box>
                </Alert>
            }

            {data.assessment?.hasDrift &&
                <Alert
                    sx={{ mb: 2 }}
                    variant="outlined"
                    severity='warning'>
                    <AlertTitle>Drift detected</AlertTitle>
                    {DRIFT_ALERT_DESCRIPTION}
                    <Box sx={{ mt: 1, display: "flex", justifyContent: "flex-end" }}>
                        <Tooltip
                            title="Creates a new apply run using the last applied configuration to reconcile the detected drift"
                            placement="bottom"
                        >
                            <Button
                                variant="outlined"
                                color="warning"
                                size="small"
                                onClick={() => setShowReconcileDialog(true)}
                            >
                                Reconcile Workspace Drift
                            </Button>
                        </Tooltip>
                    </Box>
                </Alert>
            }

            {(data.currentApplyRun || data.currentStateVersion) && <Box marginBottom={2}>
                <WorkspaceDetailsStatusPanel fragmentRef={data} />
            </Box>}

            {!data.currentStateVersion && <WorkspaceDetailsEmpty fragmentRef={data} />}

            {error && <Alert sx={{ marginTop: 2, mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            {data.currentStateVersion && <React.Fragment>
                <Box sx={{ borderBottom: 1, borderColor: 'divider', marginBottom: 2 }}>
                    <Tabs value={tab} onChange={onTabChange} variant="scrollable" scrollButtons="auto" allowScrollButtonsMobile>
                        <Tab label="Resources" value="resources" />
                        <Tab label="Input Variables" value="inputs" />
                        <Tab label="Outputs" value="outputs" />
                        <Tab label="Dependencies" value="dependencies" />
                        <Tab label="Checks" value="checks" />
                        <Tab label="Drift" value="drift" />
                        <Tab label="State File" value="stateFile" />
                    </Tabs>
                </Box>
                {tab === 'resources' && <StateVersionResources fragmentRef={data.currentStateVersion.inventory} destroyed={workspaceDestroyed} />}
                {tab === 'inputs' && <React.Fragment>
                    {data.currentStateVersion.run && <StateVersionInputVariables fragmentRef={data.currentStateVersion.run} />}
                    {!data.currentStateVersion.run && <NoResults sx={{ mt: 4 }}>
                        Input variables are not available for manually updated state versions
                    </NoResults>}
                </React.Fragment>}
                {tab === 'outputs' && <StateVersionOutputs fragmentRef={data.currentStateVersion} />}
                {tab === 'dependencies' && <StateVersionDependencies fragmentRef={data.currentStateVersion.inventory} />}
                {tab === 'checks' && <StateVersionCheckResults fragmentRef={data.currentStateVersion.inventory} />}
                {tab === 'drift' && <WorkspaceDetailsDriftDetection fragmentRef={data} />}
                {tab === 'stateFile' && <TabContent>
                    <StateVersionFile fragmentRef={data.currentStateVersion} />
                </TabContent>}
            </React.Fragment>}
            {showDestroyRunConfirmationDialog && (
                <ConfirmationDialog
                    title="Destroy Workspace"
                    maxWidth="sm"
                    confirmLabel={destroyAutoApply ? "Destroy" : "Create Destroy Plan"}
                    confirmInProgress={destroyWorkspaceIsInFlight ?? false}
                    onConfirm={() => onDestroyConfirmationDialogClosed(true)}
                    onClose={() => onDestroyConfirmationDialogClosed()}
                >
                    <Alert severity="warning">
                        <AlertTitle>Warning</AlertTitle>
                        Initiating a destroy workspace run will <strong><ins>permanently</ins></strong> destroy all resources managed by this workspace.
                        This operation will use the same module or configuration version that created the current workspace state. Any variables used in
                        the most recent successful apply operation will automatically be included.
                    </Alert>
                    <Paper variant="outlined" sx={{ mt: 2, p: 2 }}>
                        <FormControlLabel
                            sx={{ alignItems: 'flex-start', m: 0, gap: 1 }}
                            control={
                                <Switch
                                    checked={destroyAutoApply}
                                    onChange={e => setDestroyAutoApply(e.target.checked)}
                                />
                            }
                            label={
                                <Box sx={{ pt: 0.75 }}>
                                    <Typography variant="subtitle1" fontWeight={500}>Auto apply</Typography>
                                    <Typography variant="body2" color="textSecondary">
                                        Automatically apply the destroy plan once it completes successfully, without waiting for manual approval.
                                    </Typography>
                                </Box>
                            }
                        />
                    </Paper>
                </ConfirmationDialog>
            )}
            {showReconcileDialog && (
                <ConfirmationDialog
                    title="Reconcile Workspace Drift"
                    confirmLabel="Confirm"
                    confirmColor="primary"
                    confirmInProgress={isReconcileInFlight}
                    onConfirm={() => onReconcileDialogClosed(true)}
                    onClose={() => onReconcileDialogClosed()}
                >
                    Are you sure you want to reconcile the workspace drift?
                </ConfirmationDialog>
            )}
        </Box>
    );
}

export default WorkspaceDetailsIndex;
