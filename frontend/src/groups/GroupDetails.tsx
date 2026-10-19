import { Box, Button, CircularProgress, Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React, { Suspense, useEffect, useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import ConfirmationDialog from '../common/ConfirmationDialog';
import TRNButton from '../common/TRNButton';
import Variables from '../namespace/variables/Variables';
import NamespaceMemberships from '../namespace/members/NamespaceMemberships';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../skeletons/ListSkeleton';
import GroupSettings from './settings/GroupSettings'
import GroupDetailsDrawer from './GroupDetailsDrawer';
import ManagedIdentities from './managedidentity/ManagedIdentities';
import GroupRunners from './runner/GroupRunners';
import NamespaceList from './NamespaceList';
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
import CleanupPolicies from '../namespace/cleanuppolicy/CleanupPolicies';
import ProviderMirrors from '../namespace/providermirror/ProviderMirrors';
import NamespaceActivity from '../namespace/activity/NamespaceActivity';
import GroupNotificationPreference from '../notifications/GroupNotificationPreference';
import TerraformModules from './terraformmodule/TerraformModules';
import GroupPackages from './package/GroupPackages';
import GroupPolicies from '../namespace/policies/GroupPolicies';
import GroupRuns from './GroupRuns';
import NamespaceFavoriteButton from '../common/NamespaceFavoriteButton';
import { useAgentCopilot } from '../ai/AgentCopilotProvider';
import { PageLayoutProvider } from '../layout/PageLayoutContext';
import NamespaceAvatar from '../common/NamespaceAvatar';

interface Props {
    fragmentRef: GroupDetailsFragment_group$key
    route: string
}

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
            ...GroupRunsFragment_group
            ...ServiceAccountsFragment_group
            ...TerraformModulesFragment_group
            ...GroupPackagesFragment_group
            ...GroupPoliciesFragment_group
            ...VCSProvidersFragment_group
            ...FederatedRegistriesFragment_group
            ...VariablesFragment_variables
            ...NamespaceMembershipsFragment_memberships
            ...GPGKeysFragment_group
            ...ProviderMirrorsFragment_namespace
            ...CleanupPoliciesFragment_namespace
            ...NamespaceActivityFragment_activity
            ...GroupSettingsFragment_group
        }
    `, props.fragmentRef)

    const groupPath = data.fullPath;

    return (
        <Box display="flex">
            <GroupDetailsDrawer groupName={data.name} groupPath={groupPath} route={route} />
            <Box component="main" flexGrow={1} minWidth={0}>
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
                    <PageLayoutProvider>
                        <Routes>
                            <Route path={`${groupPath}/*`} element={<GroupDetailsIndex fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/activity/*`} element={<NamespaceActivity fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/runs/*`} element={<GroupRuns fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/managed_identities/*`} element={<ManagedIdentities fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/runners/*`} element={<GroupRunners fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/service_accounts/*`} element={<ServiceAccounts fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/terraform_modules/*`} element={<TerraformModules fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/packages/*`} element={<GroupPackages fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/policies/*`} element={<GroupPolicies fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/vcs_providers/*`} element={<VCSProviders fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/federated_registries/*`} element={<FederatedRegistries fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/provider_mirror/*`} element={<ProviderMirrors fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/cleanup_policies/*`} element={<CleanupPolicies fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/variables/*`} element={<Variables fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/members/*`} element={<NamespaceMemberships fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/keys/*`} element={<GPGKeys fragmentRef={data} />} />
                            <Route path={`${groupPath}/-/settings/*`} element={<GroupSettings fragmentRef={data} />} />
                        </Routes>
                    </PageLayoutProvider>
                </Suspense>
            </Box>
        </Box>
    );
}

interface GroupDetailsIndexProps {
    fragmentRef: GroupDetailsIndexFragment_group$key
}

function GroupDetailsIndex(props: GroupDetailsIndexProps) {
    const theme = useTheme();
    const [showDeleteConfirmationDialog, setShowDeleteConfirmationDialog] = useState<boolean>(false);
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();
    const { setState: setCopilotState } = useAgentCopilot();

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
                descendentGroups(first: 0) {
                    totalCount
                }
                workspaces(first: 0) {
                    totalCount
                }
                ...MigrateGroupDialogFragment_group
                ...GroupNotificationPreferenceFragment_group
            }
        `,
        props.fragmentRef
    );

    const subgroupCount = data.descendentGroups.totalCount;
    const workspaceCount = data.workspaces.totalCount;

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

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commit({
                variables: {
                    input: {
                        groupPath: data.fullPath
                    },
                },
                // Namespace lists are cached per view, search and expanded group, so mark the whole store stale and let
                // whichever list mounts next refetch instead of patching each connection
                updater: (store, deleteData) => {
                    if (!deleteData?.deleteGroup.problems.length) {
                        store.invalidateStore();
                    }
                },
                onCompleted: deleteData => {
                    setShowDeleteConfirmationDialog(false);
                    if (deleteData.deleteGroup.problems.length) {
                        enqueueSnackbar(deleteData.deleteGroup.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        if (data.fullPath.includes("/")) {
                            navigate(`../${data.fullPath.slice(0, -data.name.length - 1)}`);
                        } else {
                            navigate("..");
                        }
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

    useEffect(() => {
        setCopilotState({
            contextMessage: `The user is currently viewing the tharsis group ${data.fullPath} with ID: ${data.id}.`,
            suggestions: []
        });
        return () => {
            setCopilotState(undefined);
        }
    }, [data.fullPath, setCopilotState]);

    return (
        <Box>
            <NamespaceBreadcrumbs namespacePath={data.fullPath} />
            {<React.Fragment>
                <Box sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    [theme.breakpoints.down('lg')]: {
                        flexDirection: 'column',
                        alignItems: 'flex-start',
                        '& > *': { marginBottom: 2 },
                    }
                }}>
                    <Box display="flex" marginBottom={3} alignItems="center">
                        <NamespaceAvatar name={data.name} type="group" size="lg" sx={{ marginRight: 2 }} />
                        <Stack>
                            <Typography noWrap variant="h5" sx={{ maxWidth: 400, fontWeight: "bold" }}>{data.name}</Typography>
                            <Typography color="textSecondary" variant="subtitle2">{data.description}</Typography>
                            <Typography color="textSecondary" variant="caption">
                                {subgroupCount} subgroup{subgroupCount === 1 ? '' : 's'} · {workspaceCount} workspace{workspaceCount === 1 ? '' : 's'}
                            </Typography>
                        </Stack>
                    </Box>
                    <Box>
                        <Box sx={{ display: 'flex', gap: 1, flexDirection: { xs: 'column', md: 'row' }, alignItems: { xs: 'stretch', md: 'center' } }}>
                            <Stack direction="row" spacing={1} alignItems="center">
                                <NamespaceFavoriteButton
                                    namespacePath={data.fullPath}
                                    namespaceType="GROUP"
                                />
                                <GroupNotificationPreference fragmentRef={data} />
                                <TRNButton trn={data.metadata.trn} size="small" />
                            </Stack>
                            <Box sx={{ display: 'flex', gap: 1, flexDirection: { xs: 'column', md: 'row' } }}>
                                <Button sx={{ width: { xs: '100%', md: 'auto' } }} size="small" variant="outlined" color="info" onClick={() => (navigate(`/workspaces/-/new?parent=${data.fullPath}`))}>New Workspace</Button>
                                <Button sx={{ width: { xs: '100%', md: 'auto' } }} size="small" variant="outlined" color="info" onClick={() => (navigate(`/groups/-/new?parent=${data.fullPath}`))}>New Subgroup</Button>
                            </Box>
                        </Box>
                    </Box>
                </Box>
                <Suspense fallback={<ListSkeleton rowCount={10} />}>
                    {/* Keyed by group so the view and search start fresh when navigating to another group */}
                    <NamespaceList key={data.id} parentPath={data.fullPath} />
                </Suspense>
            </React.Fragment>}
            {showDeleteConfirmationDialog && (
                <ConfirmationDialog
                    title="Delete Group"
                    confirmLabel="Delete"
                    confirmInProgress={commitInFlight}
                    onConfirm={() => onDeleteConfirmationDialogClosed(true)}
                    onClose={() => onDeleteConfirmationDialogClosed()}
                >
                    Are you sure you want to delete group <strong>{data.name}</strong>?
                </ConfirmationDialog>
            )}
        </Box>
    );
}

export default GroupDetails;
