import CopyIcon from '@mui/icons-material/ContentCopy';
import StateIcon from '@mui/icons-material/InsertDriveFileOutlined';
import { LoadingButton } from "@mui/lab";
import { Alert, AlertTitle, Avatar, Box, Button, Chip, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, Paper, Stack, Tab, Tabs, Tooltip, Typography, useTheme } from '@mui/material';
import teal from '@mui/material/colors/teal';
import graphql from 'babel-plugin-relay/macro';
import { CubeOutline as ModuleIcon } from 'mdi-material-ui';
import React, { useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import Timestamp from '../common/Timestamp';
import TabContent from '../common/TabContent';
import TRNButton from '../common/TRNButton';
import { MutationError } from '../common/error';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import Link from '../routes/Link';
import WorkspaceDetailsCurrentJob from './WorkspaceDetailsCurrentJob';
import WorkspaceDetailsEmpty from './WorkspaceDetailsEmpty';
import { WorkspaceDetailsIndexFragment_workspace$key } from './__generated__/WorkspaceDetailsIndexFragment_workspace.graphql';
import { WorkspaceDetailsIndex_DestroyWorkspaceMutation } from './__generated__/WorkspaceDetailsIndex_DestroyWorkspaceMutation.graphql';
import RunStatusChip from './runs/RunStatusChip';
import StateVersionDependencies from './state/StateVersionDependencies';
import StateVersionFile from './state/StateVersionFile';
import StateVersionInputVariables from './state/StateVersionInputVariables';
import StateVersionOutputs from './state/StateVersionOutputs';
import StateVersionResources from './state/StateVersionResources';
import WorkspaceDetailsDriftDetection from './WorkspaceDetailsDriftDetection';
import WorkspaceNotificationPreference from '../notifications/WorkspaceNotificationPreference';

const DRIFT_ALERT_DESCRIPTION = "This workspace has drifted from its configuration; this can happen if the resources were modified outside of Tharsis, or if the infrastructure was changed directly through the cloud provider console."

interface Props {
    fragmentRef: WorkspaceDetailsIndexFragment_workspace$key
}

interface ConfirmationDialogProps {
    open: boolean
    onClose: (confirm?: boolean) => void
    deleteInProgress: boolean | undefined
}

function DestroyRunConfirmationDialog({ deleteInProgress, onClose, open }: ConfirmationDialogProps) {

    return (
        <Dialog
            keepMounted
            maxWidth="sm"
            open={open}
        >
            <DialogTitle>Destroy Workspace</DialogTitle>
            <DialogContent >
                <Alert sx={{ mb: 2 }} severity="warning">
                    <AlertTitle>Warning</AlertTitle>
                    Initiating a destroy workspace run will <strong><ins>permanently</ins></strong> destroy all resources managed by this workspace.
                    This operation will use the same module or configuration version that created the current workspace state. Any variables used in
                    the most recent successful apply operation will automatically be included. The created plan will have to be applied manually.
                </Alert>
            </DialogContent>
            <DialogActions>
                <Button
                    color="inherit"
                    onClick={() => onClose()}>Cancel</Button>
                <LoadingButton
                    color="error"
                    variant="outlined"
                    loading={deleteInProgress}
                    onClick={() => onClose(true)}>Destroy</LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

function WorkspaceDetailsIndex(props: Props) {
    const { fragmentRef } = props;
    const theme = useTheme();
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const [showDestroyRunConfirmationDialog, setShowDestroyRunConfirmationDialog] = useState<boolean>(false);
    const [error, setError] = useState<MutationError>();

    const tab = searchParams.get('tab') ?? 'resources';

    const data = useFragment<WorkspaceDetailsIndexFragment_workspace$key>(
        graphql`
      fragment WorkspaceDetailsIndexFragment_workspace on Workspace
      {
        id
        name
        description
        fullPath
        preventDestroyPlan
        metadata {
            trn
        }
        assessment {
            hasDrift
        }
        ...WorkspaceDetailsEmptyFragment_workspace
        ...WorkspaceDetailsCurrentJobFragment_workspace
        ...WorkspaceNotificationPreferenceFragment_workspace
        currentJob {
            id
        }
        currentStateVersion {
            id
            ...StateVersionOutputsFragment_outputs
            ...StateVersionResourcesFragment_resources
            ...StateVersionDependenciesFragment_dependencies
            ...StateVersionFileFragment_stateVersion
            metadata {
                createdAt
            }
            run {
                ...StateVersionInputVariablesFragment_variables
                id
                status
                createdBy
                isDestroy
                moduleSource
                moduleVersion
                metadata {
                    createdAt
                }
                configurationVersion {
                    id
                    vcsEvent {
                        status
                    }
                }
                plan {
                    status
                    metadata {
                        createdAt
                    }
                }
                apply {
                    status
                    triggeredBy
                    metadata {
                        createdAt
                        updatedAt
                    }
                }
            }
        }
        ...WorkspaceDetailsDriftDetectionFragment_workspace
      }
    `, fragmentRef);

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
    };

    const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
        navigate({
            search: `?tab=${newValue}`
        }, {
            replace: true
        });
    };

    return (
        <Box>
            <NamespaceBreadcrumbs namespacePath={data.fullPath} />
            <Box
                sx={{
                    display: "flex",
                    justifyContent: "space-between",
                    mb: 2,
                    [theme.breakpoints.down('lg')]: {
                        flexDirection: 'column',
                        alignItems: 'flex-start',
                        '& > *:not(:last-child)': { mb: 2 },
                    }
                }}>
                <Box display="flex" alignItems="center">
                    <Avatar sx={{ width: 56, height: 56, marginRight: 2, bgcolor: teal[200] }} variant="rounded">{data.name[0].toUpperCase()}</Avatar>
                    <Stack>
                        <Typography variant="h5" sx={{ fontWeight: "bold" }}>{data.name}</Typography>
                        <Typography color="textSecondary" variant="subtitle2">{data.description}</Typography>
                    </Stack>
                </Box>
                <Stack direction="row" spacing={1}>
                    <WorkspaceNotificationPreference fragmentRef={data} />
                    <TRNButton trn={data.metadata.trn} size="small" />
                    {(data.currentStateVersion && data.currentStateVersion.run) && (
                        <Box>
                            <Tooltip
                                title={data.preventDestroyPlan ? "Prevent Destroy Run is enabled for this workspace." : "Create a destroy run, which destroys all resources in this workspace."}
                                placement="top"
                            >
                                <span>
                                    <Button
                                        size="small"
                                        variant="outlined"
                                        color="error"
                                        disabled={data.preventDestroyPlan}
                                        onClick={() => setShowDestroyRunConfirmationDialog(true)}
                                    >
                                        Destroy Workspace
                                    </Button>
                                </span>
                            </Tooltip>
                        </Box>
                    )}
                </Stack>
            </Box>

            {data.assessment?.hasDrift &&
                <Alert
                    sx={{ mb: 2 }}
                    variant="outlined"
                    severity='warning'>
                    <AlertTitle>Drift detected</AlertTitle>
                    {DRIFT_ALERT_DESCRIPTION}
                </Alert>
            }

            {data.currentJob && <Box marginBottom={2}>
                <WorkspaceDetailsCurrentJob fragmentRef={data} />
            </Box>}

            {!data.currentStateVersion && <WorkspaceDetailsEmpty fragmentRef={data} />}

            {error && <Alert sx={{ marginTop: 2, mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            {data.currentStateVersion && <Paper sx={{ marginBottom: 2, padding: 2 }}>
                <Box display="flex" justifyContent="space-between" alignItems="center">
                    <Stack direction="row" spacing={2}>
                        <StateIcon />
                        <Typography component="div">
                            State last updated{' '}
                            <Timestamp component="span" timestamp={data.currentStateVersion.metadata.createdAt} />
                            {' '}
                            {!data.currentStateVersion.run && 'by manual update'}
                            {data.currentStateVersion.run && <React.Fragment>
                                by run{' '}
                                <Link to={`/groups/${data.fullPath}/-/runs/${data.currentStateVersion.run.id}`}>
                                    {data.currentStateVersion.run.id.substring(0, 8)}...
                                </Link>
                            </React.Fragment>}
                        </Typography>
                    </Stack>
                    {data.currentStateVersion.run && <React.Fragment>
                        <RunStatusChip
                            to={`/groups/${data.fullPath}/-/runs/${data.currentStateVersion.run.id}`}
                            status={data.currentStateVersion.run.status}
                        />
                    </React.Fragment>}
                </Box>
            </Paper>}

            {data.currentStateVersion?.run?.moduleSource &&
                <Paper sx={{ marginBottom: 2, padding: 2 }}>
                    <Stack direction="row" spacing={2}>
                        <ModuleIcon />
                        <Stack direction="row" spacing={1} alignItems="center">
                            <Typography color="textSecondary">Module:</Typography>
                            <Typography sx={{ wordBreak: 'break-all' }}>
                                {data.currentStateVersion.run.moduleSource}
                            </Typography>
                            <IconButton sx={{ padding: 0 }} onClick={() => navigator.clipboard.writeText(data.currentStateVersion?.run?.moduleSource ?? '')}>
                                <CopyIcon sx={{ width: 16, height: 16 }} />
                            </IconButton>
                        </Stack>
                        <Stack direction="row" spacing={1} alignItems="center">
                            <Typography color="textSecondary">Version:</Typography>
                            <Chip size="small" label={data.currentStateVersion.run.moduleVersion} />
                        </Stack>
                    </Stack>
                </Paper>}

            {data.currentStateVersion?.run?.configurationVersion &&
                <Paper sx={{ marginBottom: 2, padding: 2 }}>
                    <Stack direction="row" spacing={2}>
                        <ModuleIcon />
                        <Stack direction="row" spacing={1} alignItems="center">
                            <Typography color="textSecondary">Configuration Version:</Typography>
                            <Typography sx={{ wordBreak: 'break-all' }}>
                                {data.currentStateVersion.run.configurationVersion.id.substring(0, 8)}...
                            </Typography>
                            <IconButton sx={{ padding: 0 }} onClick={() => navigator.clipboard.writeText(data.currentStateVersion?.run?.configurationVersion?.id ?? '')}>
                                <CopyIcon sx={{ width: 16, height: 16 }} />
                            </IconButton>
                        </Stack>
                    </Stack>
                </Paper>}

            {data.currentStateVersion && <React.Fragment>
                <Box sx={{ borderBottom: 1, borderColor: 'divider', marginBottom: 2 }}>
                    <Tabs value={tab} onChange={onTabChange}>
                        <Tab label="Resources" value="resources" />
                        <Tab label="Input Variables" value="inputs" />
                        <Tab label="Outputs" value="outputs" />
                        <Tab label="Dependencies" value="dependencies" />
                        <Tab label="Drift" value="drift" />
                        <Tab label="State File" value="stateFile" />
                    </Tabs>
                </Box>
                {tab === 'resources' && <StateVersionResources fragmentRef={data.currentStateVersion} />}
                {tab === 'inputs' && <React.Fragment>
                    {data.currentStateVersion.run && <StateVersionInputVariables fragmentRef={data.currentStateVersion.run} />}
                    {!data.currentStateVersion.run && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                        <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                            <Typography color="textSecondary" align="center">
                                Input variables are not available for manually updated state versions
                            </Typography>
                        </Box>
                    </Paper>}
                </React.Fragment>}
                {tab === 'outputs' && <StateVersionOutputs fragmentRef={data.currentStateVersion} />}
                {tab === 'dependencies' && <StateVersionDependencies fragmentRef={data.currentStateVersion} />}
                {tab === 'drift' && <WorkspaceDetailsDriftDetection fragmentRef={data} />}
                {tab === 'stateFile' && <TabContent>
                    <StateVersionFile fragmentRef={data.currentStateVersion} />
                </TabContent>}
            </React.Fragment>}
            <DestroyRunConfirmationDialog
                open={showDestroyRunConfirmationDialog}
                deleteInProgress={destroyWorkspaceIsInFlight}
                onClose={onDestroyConfirmationDialogClosed}
            />
        </Box>
    );
}

export default WorkspaceDetailsIndex;
