import { TabContext, TabList } from '@mui/lab';
import MuiTabPanel, { TabPanelProps } from '@mui/lab/TabPanel';
import { LoadingButton } from '@mui/lab';
import { Avatar, Button, CircularProgress, Stack, styled, Typography } from '@mui/material';
import { Box, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import teal from '@mui/material/colors/teal';
import { TabProps } from '@mui/material/Tab';
import graphql from 'babel-plugin-relay/macro';
import React, { Suspense, useEffect, useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { LinkProps, Route, Routes } from 'react-router-dom';
import { GroupIcon, WorkspaceIcon } from '../common/Icons';
import TRNButton from '../common/TRNButton';
import Variables from '../namespace/variables/Variables';
import NamespaceMemberships from '../namespace/members/NamespaceMemberships';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import TabLink from '../routes/TabLink';
import ListSkeleton from '../skeletons/ListSkeleton';
import GroupSettings from './settings/GroupSettings'
import GroupDetailsDrawer from './GroupDetailsDrawer';
import GroupList from './GroupList';
import ManagedIdentities from './managedidentity/ManagedIdentities';
import GroupRunners from './runner/GroupRunners';
import WorkspaceList from './WorkspaceList';
import { GroupDetailsFragment_group$key } from './__generated__/GroupDetailsFragment_group.graphql';
import { GroupDetailsIndexFragment_group$key } from './__generated__/GroupDetailsIndexFragment_group.graphql';
import { GroupDetailsDeleteMutation } from './__generated__/GroupDetailsDeleteMutation.graphql';
import { useSnackbar } from 'notistack';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '@mui/material/styles';
import ServiceAccounts from './serviceaccount/ServiceAccounts';
import VCSProviders from './vcsprovider/VCSProviders';
import FederatedRegistries from './federatedregistry/FederatedRegistries';
import GPGKeys from './keys/GPGKeys';
import NamespaceActivity from '../namespace/activity/NamespaceActivity';
import GroupNotificationPreference from '../notifications/GroupNotificationPreference';

const TABS = ['workspaces', 'subgroups'];

interface Props {
    fragmentRef: GroupDetailsFragment_group$key
    route: string
}

interface ConfirmationDialogProps {
    groupName: string
    deleteInProgress: boolean;
    keepMounted: boolean;
    open: boolean;
    onClose: (confirm?: boolean) => void
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { groupName, deleteInProgress, onClose, open, ...other } = props;
    return (
        <Dialog
            maxWidth="xs"
            open={open}
            {...other}
        >
            <DialogTitle>Delete Group</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete group <strong>{groupName}</strong>?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>Cancel</Button>
                <LoadingButton color="error" loading={deleteInProgress} onClick={() => onClose(true)}>Delete</LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

const Tab = styled(TabLink)<TabProps | LinkProps>(() => ({
    textTransform: "none",
    minHeight: 48
}));

const TabPanel = styled(MuiTabPanel)<TabPanelProps>(() => ({
    padding: 1
}));

function GroupDetails(props: Props) {
    const { route } = props;
    const data = useFragment<GroupDetailsFragment_group$key>(
        graphql`
        fragment GroupDetailsFragment_group on Group
        {
            id
            fullPath
            name
            ...GroupDetailsIndexFragment_group
            ...ManagedIdentitiesFragment_group
            ...GroupRunnersFragment_group
            ...ServiceAccountsFragment_group
            ...VCSProvidersFragment_group
            ...FederatedRegistriesFragment_group
            ...VariablesFragment_variables
            ...NamespaceMembershipsFragment_memberships
            ...GPGKeysFragment_group
            ...NamespaceActivityFragment_activity
            ...GroupSettingsFragment_group
        }
    `, props.fragmentRef)

    const groupPath = data.fullPath;

    return (
        <Box display="flex">
            <GroupDetailsDrawer groupName={data.name} groupPath={groupPath} route={route} />
            <Box component="main" flexGrow={1}>
                <Suspense fallback={
                    <Box
                        sx={{
                            width: '100%',
                            height: `calc(100vh - 64px)`,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center'
                        }}
                    >
                        <CircularProgress />
                    </Box>}>
                    <Box maxWidth={1200} margin="auto" padding={2}>
                        <Routes>
                            <Route path={`${groupPath}/*`} element={<GroupDetailsIndex route={route} fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/activity/*`} element={<NamespaceActivity fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/managed_identities/*`} element={<ManagedIdentities fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/runners/*`} element={<GroupRunners fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/service_accounts/*`} element={<ServiceAccounts fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/vcs_providers/*`} element={<VCSProviders fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/federated_registries/*`} element={<FederatedRegistries fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/variables/*`} element={<Variables fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/members/*`} element={<NamespaceMemberships fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/keys/*`} element={<GPGKeys fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/settings/*`} element={<GroupSettings fragmentRef={data} />} />
                        </Routes>
                    </Box>
                </Suspense>
            </Box>
        </Box>
    );
}

interface GroupDetailsIndexProps {
    fragmentRef: GroupDetailsIndexFragment_group$key
    route: string
}

function GroupDetailsIndex(props: GroupDetailsIndexProps) {
    const { route } = props;
    const theme = useTheme();
    const [tab, setTab] = useState('');
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();

    const data = useFragment<GroupDetailsIndexFragment_group$key>(
        graphql`
            fragment GroupDetailsIndexFragment_group on Group {
                id
                name
                description
                fullPath
                metadata {
                    trn
                }
                workspaces(first: 0) {
                    totalCount
                }
                descendentGroups(first: 0) {
                    totalCount
                }
                ...WorkspaceListFragment_group
                ...MigrateGroupDialogFragment_group
                ...GroupNotificationPreferenceFragment_group
            }
        `,
        props.fragmentRef
    );

    const [commit, commitInFlight] = useMutation<GroupDetailsDeleteMutation>(
        graphql`
            mutation GroupDetailsDeleteMutation($input:
                DeleteGroupInput!){deleteGroup(input: $input) {
                    problems {
                        message
                        field
                        type
                    }
                }
            }
        `
    );

    useEffect(() => {
        if (TABS.includes(route)) {
            setTab(route);
        } else {
            const workspaceCount = data.workspaces.totalCount ?? 0;
            const subgroupCount = data.descendentGroups.totalCount ?? 0;
            setTab(workspaceCount > 0 ? 'workspaces' : (subgroupCount > 0 ? 'subgroups' : 'workspaces'));
        }
    }, [route, data]);

    const handleTabChange = (event: React.SyntheticEvent, newValue: string) => {
        setTab(newValue);
    };

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commit({
                variables: {
                    input: {
                        groupPath: data.fullPath
                    },
                },
                onCompleted: deleteData => {
                    setShowDeleteConfirmationDialog(false);
                    if (deleteData.deleteGroup.problems.length) {
                        enqueueSnackbar(deleteData.deleteGroup.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        data.fullPath.includes("/") ?
                            navigate(`../${data.fullPath.slice(0, -data.name.length - 1)}`) : navigate("..")
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

    return (
        <Box>
            <NamespaceBreadcrumbs namespacePath={data.fullPath} />
            {(!route || TABS.includes(route)) && <React.Fragment>
                <Box sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    [theme.breakpoints.down('lg')]: {
                        flexDirection: 'column',
                        alignItems: 'flex-start',
                        '& > *': { marginBottom: 2 },
                    }
                }}>
                    <Box display="flex" marginBottom={4} alignItems="center">
                        <Avatar sx={{ width: 56, height: 56, marginRight: 2, bgcolor: teal[200] }} variant="rounded">{data.name[0].toUpperCase()}</Avatar>
                        <Stack>
                            <Typography noWrap variant="h5" sx={{ maxWidth: 400, fontWeight: "bold" }}>{data.name}</Typography>
                            <Typography color="textSecondary" variant="subtitle2">{data.description}</Typography>
                        </Stack>
                    </Box>
                    <Box>
                        <Stack direction="row" spacing={1}>
                            <GroupNotificationPreference fragmentRef={data} />
                            <TRNButton trn={data.metadata.trn} size="small" />
                            <Button size="small" variant="outlined" color="info" onClick={() => (navigate(`/workspaces/-/new?parent=${data.fullPath}`))}>New Workspace</Button>
                            <Button size="small" variant="outlined" color="info" onClick={() => (navigate(`/groups/-/new?parent=${data.fullPath}`))}>New Subgroup</Button>
                        </Stack>
                    </Box>
                </Box>
                {tab && <TabContext value={tab}>
                    <Box sx={{ border: 1, borderColor: 'divider' }}>
                        <TabList onChange={handleTabChange} aria-label="group tabs" sx={{ minHeight: 0 }}>
                            <Tab
                                icon={<WorkspaceIcon />}
                                iconPosition="start"
                                label={`${data.workspaces.totalCount} Workspace${data.workspaces.totalCount === 1 ? '' : 's'}`}
                                value="workspaces"
                                to={`/groups/${data.fullPath}/-/workspaces`}
                            />
                            <Tab
                                icon={<GroupIcon />}
                                iconPosition="start"
                                label={`${data.descendentGroups.totalCount} Subgroup${data.descendentGroups.totalCount === 1 ? '' : 's'}`}
                                value="subgroups"
                                to={`/groups/${data.fullPath}/-/subgroups`}
                            />
                        </TabList>
                    </Box>
                    <TabPanel value="workspaces">
                        {data.workspaces.totalCount > 0 && <Suspense fallback={<ListSkeleton rowCount={10} />}>
                            <WorkspaceList fragmentRef={data}/>
                        </Suspense>}
                        {data.workspaces.totalCount === 0 && <Box padding={4} display="flex" justifyContent="center" alignItems="center">
                            <Typography color="textSecondary">No workspaces in this group</Typography>
                        </Box>}
                    </TabPanel>
                    <TabPanel value="subgroups">
                        {data.descendentGroups.totalCount > 0 && <Suspense fallback={<ListSkeleton rowCount={10} />}>
                            <GroupList groupPath={data.fullPath} />
                        </Suspense>}
                        {data.descendentGroups.totalCount === 0 && <Box padding={4} display="flex" justifyContent="center" alignItems="center">
                            <Typography color="textSecondary">No subgroups in this group</Typography>
                        </Box>}
                    </TabPanel>
                </TabContext>}
            </React.Fragment>}
            <DeleteConfirmationDialog
                groupName={data.name}
                keepMounted
                deleteInProgress={commitInFlight}
                open={showDeleteConfirmationDialog}
                onClose={onDeleteConfirmationDialogClosed}
            />
        </Box>
    );
}

export default GroupDetails;
