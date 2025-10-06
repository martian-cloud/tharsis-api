import { ArrowDropUp } from '@mui/icons-material';
import { default as ArrowDropDown, default as ArrowDropDownIcon } from '@mui/icons-material/ArrowDropDown';
import { LoadingButton } from '@mui/lab';
import { Alert, Box, Button, ButtonGroup, Chip, Collapse, Dialog, DialogActions, DialogContent, DialogTitle, Link, Menu, MenuItem, Stack, Tab, Tabs, Typography, useTheme } from "@mui/material";
import { green } from "@mui/material/colors";
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { useSnackbar } from 'notistack';
import { useMemo, useState } from "react";
import { useFragment, useMutation, useSubscription } from 'react-relay/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { ConnectionHandler, ConnectionInterface, GraphQLSubscriptionConfig, RecordSourceProxy } from "relay-runtime";
import { RunnerIcon } from "../common/Icons";
import TabContent from "../common/TabContent";
import TRNButton from "../common/TRNButton";
import AssignedServiceAccountList from "./AssignedServiceAccountList";
import RunnerChip from "./RunnerChip";
import RunnerJobList from "./RunnerJobList";
import RunnerSessionList, { GetConnections as GetRunnerSessionConnections } from "./RunnerSessionList";
import { RunnerDetailsDeleteMutation } from "./__generated__/RunnerDetailsDeleteMutation.graphql";
import { RunnerDetailsFragment_runner$key } from "./__generated__/RunnerDetailsFragment_runner.graphql";
import { RunnerDetailsSessionEventsSubscription, RunnerDetailsSessionEventsSubscription$data } from "./__generated__/RunnerDetailsSessionEventsSubscription.graphql";

const runnerSessionEventsSubscription = graphql`subscription RunnerDetailsSessionEventsSubscription($input: RunnerSessionEventSubscriptionInput!) {
    runnerSessionEvents(input: $input) {
      action
      runnerSession {
        id
        ...RunnerSessionListItemFragment
      }
    }
  }`;

interface ConfirmationDialogProps {
    runnerName: string
    deleteInProgress: boolean;
    keepMounted: boolean;
    open: boolean;
    onClose: (confirm?: boolean) => void
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { runnerName, deleteInProgress, onClose, open, ...other } = props;

    return (
        <Dialog
            maxWidth="xs"
            open={open}
            {...other}
        >
            <DialogTitle>Delete Runner</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete runner <strong>{runnerName}</strong>?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton color="error" loading={deleteInProgress} onClick={() => onClose(true)}>
                    Delete
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

interface Props {
    fragmentRef: RunnerDetailsFragment_runner$key
    getConnections: () => [string]
}

function RunnerDetails({ fragmentRef, getConnections }: Props) {
    const [showMore, setShowMore] = useState(false);
    const [menuAnchorEl, setMenuAnchorEl] = useState<Element | null>(null);
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const [searchParams, setSearchParams] = useSearchParams();
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();
    const tab = searchParams.get('tab') || 'details';
    const theme = useTheme();

    const runner = useFragment<RunnerDetailsFragment_runner$key>(graphql`
        fragment RunnerDetailsFragment_runner on Runner
        {
            id
            name
            type
            disabled
            description
            createdBy
            tags
            runUntaggedJobs
            metadata {
                createdAt
                trn
            }
            assignedServiceAccounts (first: 0) {
                totalCount
            }
            sessions(first: 1, sort: LAST_CONTACTED_AT_DESC) {
                edges {
                    node {
                        active
                        lastContacted
                    }
                }
            }
            ...AssignedServiceAccountListFragment_runner
        }
    `, fragmentRef);

    const [commit, commitInFlight] = useMutation<RunnerDetailsDeleteMutation>(graphql`
            mutation RunnerDetailsDeleteMutation($input: DeleteRunnerInput!, $connections: [ID!]!) {
                deleteRunner(input: $input) {
                    runner {
                        id @deleteEdge(connections: $connections)
                    }
                    problems {
                        message
                        field
                        type
                    }
                }
            }
        `);

    const runnerSessionsSubscriptionConfig = useMemo<GraphQLSubscriptionConfig<RunnerDetailsSessionEventsSubscription>>(() => ({
        variables: { input: { runnerId: runner.id } },
        subscription: runnerSessionEventsSubscription,
        onCompleted: () => console.log("Subscription completed"),
        onError: () => console.warn("Subscription error"),
        updater: (store: RecordSourceProxy, payload: RunnerDetailsSessionEventsSubscription$data | null | undefined) => {
            if (!payload) {
                return;
            }
            const record = store.get(payload.runnerSessionEvents.runnerSession.id);
            if (record == null) {
                return;
            }
            GetRunnerSessionConnections(runner.id).forEach(id => {
                const connectionRecord = store.get(id);
                if (connectionRecord) {
                    const { NODE, EDGES } = ConnectionInterface.get();

                    const recordId = record.getDataID();
                    // Check if edge already exists in connection
                    const nodeAlreadyExistsInConnection = connectionRecord
                        .getLinkedRecords(EDGES)
                        ?.some(
                            edge => edge?.getLinkedRecord(NODE)?.getDataID() === recordId,
                        );
                    if (!nodeAlreadyExistsInConnection) {
                        const totalCount = connectionRecord.getValue('totalCount') as number;
                        connectionRecord.setValue(totalCount + 1, 'totalCount');

                        // Create Edge
                        const edge = ConnectionHandler.createEdge(
                            store,
                            connectionRecord,
                            record,
                            'RunnerSessionEdge'
                        );
                        if (edge) {
                            // Add edge to the beginning of the connection
                            ConnectionHandler.insertEdgeBefore(
                                connectionRecord,
                                edge,
                            );
                        }
                    }
                }
            });
        }
    }), [runner.id]);

    useSubscription<RunnerDetailsSessionEventsSubscription>(runnerSessionsSubscriptionConfig);

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commit({
                variables: {
                    input: {
                        id: runner.id
                    },
                    connections: getConnections()
                },
                onCompleted: data => {
                    setShowDeleteConfirmationDialog(false);

                    if (data.deleteRunner.problems.length) {
                        enqueueSnackbar(data.deleteRunner.problems.map((problem: any) => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        navigate(`..`);
                    }
                },
                onError: error => {
                    setShowDeleteConfirmationDialog(false);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setShowDeleteConfirmationDialog(false);
        }
    };

    const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
        searchParams.set('tab', newValue);
        setSearchParams(searchParams, { replace: true });
    };

    const onOpenMenu = (event: React.MouseEvent<HTMLButtonElement>) => {
        setMenuAnchorEl(event.currentTarget);
    };

    const onMenuClose = () => {
        setMenuAnchorEl(null);
    };

    const onMenuAction = (actionCallback: () => void) => {
        setMenuAnchorEl(null);
        actionCallback();
    };

    const lastContacted = runner.sessions.edges?.[0]?.node?.lastContacted;
    const active = runner.sessions.edges?.[0]?.node?.active;
    const showBanner = !active && runner.assignedServiceAccounts.totalCount === 0;

    return (
        <Box>
            {showBanner &&
                <Alert sx={{ mb: 2, mt: 2 }} severity="warning" variant="outlined">No service accounts are assigned to this runner.
                </Alert>}
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                [theme.breakpoints.down('sm')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *': { mb: 2 },
                }
            }}>
                <Box display="flex" alignItems="center" mb={2}>
                    <RunnerIcon sx={{ color: active ? green[400] : null, mr: 2 }} />
                    <Box>
                        <Typography variant="h5">
                            {runner.name}
                        </Typography>
                        <Typography color="textSecondary">{runner.description}</Typography>
                        {lastContacted && <Typography variant="caption" color="textSecondary">
                            {`last seen ${moment(lastContacted as moment.MomentInput).fromNow()}`}
                        </Typography>}
                    </Box>
                </Box>
                <Box>
                    <Stack direction="row" spacing={1}>
                        <TRNButton trn={runner.metadata.trn} />
                        <ButtonGroup variant="outlined" color="primary">
                            <Button onClick={() => navigate('edit')}>Edit</Button>
                            <Button
                                color="primary"
                                size="small"
                                aria-label="more options menu"
                                aria-haspopup="menu"
                                onClick={onOpenMenu}
                            >
                                <ArrowDropDownIcon fontSize="small" />
                            </Button>
                        </ButtonGroup>
                        <Menu
                            id="runner-more-options-menu"
                            anchorEl={menuAnchorEl}
                            open={Boolean(menuAnchorEl)}
                            onClose={onMenuClose}
                            anchorOrigin={{
                                vertical: 'bottom',
                                horizontal: 'right',
                            }}
                            transformOrigin={{
                                vertical: 'top',
                                horizontal: 'right',
                            }}
                        >
                            <MenuItem onClick={() => onMenuAction(() => setShowDeleteConfirmationDialog(true))}>
                                Delete Runner
                            </MenuItem>
                        </Menu>
                    </Stack>
                </Box>
            </Box>
            <Box sx={{ display: "flex", border: 1, borderColor: 'divider', borderTopLeftRadius: 4, borderTopRightRadius: 4 }}>
                <Tabs value={tab} onChange={onTabChange}>
                    <Tab label="Details" value="details" />
                    <Tab label="Sessions" value="sessions" />
                    <Tab label="Jobs" value="jobs" />
                    {runner.type === 'group' && <Tab label="Assigned Service Accounts" value="assignedServiceAccounts" />}
                </Tabs>
            </Box>
            <TabContent>
                {tab === 'details' && <Box sx={{ border: 1, borderTop: 0, borderBottomLeftRadius: 4, borderBottomRightRadius: 4, borderColor: 'divider', padding: 2 }}>
                    <Typography mb={0.25}>Type</Typography>
                    <Typography mb={2} color="textSecondary">{runner.type[0].toUpperCase() + runner.type.slice(1).toLowerCase()}</Typography>
                    <Box mb={2}>
                        <Typography mb={1}>Tags</Typography>
                        {runner.tags.length > 0 && <Stack direction="row" spacing={1}>
                            {runner.tags.map(tag => <Chip key={tag} size="small" color="secondary" label={tag} />)}
                        </Stack>}
                        {runner.tags.length === 0 && <Typography color="textSecondary">No Tags</Typography>}
                    </Box>
                    <Box mb={2}>
                        <Typography mb={1}>Run Untagged Jobs</Typography>
                        <Typography color="textSecondary">{runner.runUntaggedJobs ? 'Yes' : 'No'}</Typography>
                    </Box>
                    <Typography mb={0.25}>Status</Typography>
                    <RunnerChip disabled={runner.disabled} />
                    <Box mt={4}>
                        <Link
                            sx={{ display: "flex", alignItems: "center" }}
                            component="button" variant="body1"
                            color="textSecondary"
                            underline="hover"
                            onClick={() => setShowMore(!showMore)}
                        >
                            More Details {showMore ? <ArrowDropUp /> : <ArrowDropDown />}
                        </Link>
                        <Collapse in={showMore} timeout="auto" unmountOnExit>
                            <Box mt={2}>
                                <Typography variant="body2">
                                    Created {moment(runner.metadata.createdAt as moment.MomentInput).fromNow()} by {runner.createdBy}
                                </Typography>
                            </Box>
                        </Collapse>
                    </Box>
                </Box>}
                {tab === 'sessions' && <RunnerSessionList />}
                {tab === 'jobs' && <RunnerJobList />}
                {tab === 'assignedServiceAccounts' && runner.type === 'group' && <AssignedServiceAccountList fragmentRef={runner} />}
            </TabContent>
            <DeleteConfirmationDialog
                runnerName={runner.name}
                keepMounted
                deleteInProgress={commitInFlight}
                open={showDeleteConfirmationDialog}
                onClose={onDeleteConfirmationDialogClosed}
            />
        </Box>
    );
}

export default RunnerDetails
