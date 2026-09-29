import NoResults from '@/common/NoResults';
import { Alert, Box, Button, CircularProgress, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { Suspense, useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { MutationError } from '../../common/error';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import AssignedManagedIdentityListItem from './AssignedManagedIdentityListItem';
import ManagedIdentityAutocomplete, { ManagedIdentityOption } from './ManagedIdentityAutocomplete';
import { AssignedManagedIdentityListFragment_workspace$key } from './__generated__/AssignedManagedIdentityListFragment_workspace.graphql';
import { AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities$key } from './__generated__/AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities.graphql';
import { AssignedManagedIdentityListQuery } from './__generated__/AssignedManagedIdentityListQuery.graphql';
import { AssignedManagedIdentityListMutation } from './__generated__/AssignedManagedIdentityListMutation.graphql';
import { AssignedManagedIdentityListUnassignMutation } from './__generated__/AssignedManagedIdentityListUnassignMutation.graphql';

interface Props {
    fragmentRef: AssignedManagedIdentityListFragment_workspace$key
}

const query = graphql`
    query AssignedManagedIdentityListQuery($id: String!) {
        node(id: $id) {
            ...on Workspace {
                ...AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities
            }
        }
    }
`;

function AssignedManagedIdentityList(props: Props) {
    const workspace = useFragment<AssignedManagedIdentityListFragment_workspace$key>(graphql`
        fragment AssignedManagedIdentityListFragment_workspace on Workspace {
            id
            fullPath
        }
    `, props.fragmentRef)

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={workspace.fullPath}
                childRoutes={[
                    { title: "managed identities", path: 'managed_identities' }
                ]}
            />
            <Typography variant="h5" gutterBottom>Assigned Managed Identities</Typography>
            <Suspense fallback={
                <Box padding={4} display="flex" justifyContent="center" alignItems="center">
                    <CircularProgress />
                </Box>
            }>
                <AssignedManagedIdentityListContent workspaceId={workspace.id} workspacePath={workspace.fullPath} />
            </Suspense>
        </Box>
    );
}

interface AssignedManagedIdentityListContentProps {
    workspaceId: string
    workspacePath: string
}

function AssignedManagedIdentityListContent(props: AssignedManagedIdentityListContentProps) {
    const { workspaceId, workspacePath } = props;
    const [selected, setSelected] = useState<ManagedIdentityOption | null>(null);
    const [error, setError] = useState<MutationError | null>()
    const { enqueueSnackbar } = useSnackbar();

    const queryData = useLazyLoadQuery<AssignedManagedIdentityListQuery>(
        query,
        { id: workspaceId },
        { fetchPolicy: 'store-and-network' },
    );

    const data = useFragment<AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities$key>(graphql`
        fragment AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities on Workspace {
            managedIdentities(includeInherited: true, first: 1) {
                edges {
                    node {
                        id
                    }
                }
            }
            assignedManagedIdentities {
                id
                ...AssignedManagedIdentityListItemFragment_managedIdentity
            }
        }
    `, queryData.node)

    const [commitAssign, assignCommitInFlight] = useMutation<AssignedManagedIdentityListMutation>(graphql`
        mutation AssignedManagedIdentityListMutation($input: AssignManagedIdentityInput!) {
            assignManagedIdentity(input: $input) {
                workspace {
                    ...AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [commitUnassign] = useMutation<AssignedManagedIdentityListUnassignMutation>(graphql`
        mutation AssignedManagedIdentityListUnassignMutation($input: AssignManagedIdentityInput!) {
            unassignManagedIdentity(input: $input) {
                workspace {
                    ...AssignedManagedIdentityListAssignedFragment_assignedManagedIdentities
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const onManagedIdentitySelected = (value: ManagedIdentityOption | null) => {
        setSelected(value);
    };

    const assignManagedIdentity = () => {
        setError(null);
        if (selected) {
            commitAssign({
                variables: {
                    input: {
                        managedIdentityId: selected?.id,
                        workspacePath: workspacePath
                    },
                },
                onCompleted: data => {
                    setSelected(null);
                    if (data.assignManagedIdentity.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.assignManagedIdentity.problems.map(problem => problem.message).join('; ')
                        });
                    }
                },
                onError: error => {
                    setSelected(null);
                    setError({
                        severity: 'error',
                        message: `Unexpected Error Occurred: ${error.message}`
                    });
                }
            })
        }
    }

    const onUnassign = (id: string) => {
        commitUnassign({
            variables: {
                input: {
                    managedIdentityId: id,
                    workspacePath: workspacePath
                },
            },
            onCompleted: data => {
                if (data.unassignManagedIdentity.problems.length) {
                    enqueueSnackbar(
                        data.unassignManagedIdentity.problems.map(problem => problem.message).join('; '),
                        { variant: 'warning' }
                    );
                }
            },
            onError: error => {
                console.log(`Error occurred ${error.message}`);
                enqueueSnackbar(
                    error.message,
                    { variant: 'error' }
                );
            }
        })
    };

    const assignedManagedIdentities = data?.assignedManagedIdentities ?? [];

    const assignedManagedIdentityIds = assignedManagedIdentities.reduce((accumulator, item) => {
        accumulator.add(item.id);
        return accumulator;
    }, new Set());

    const edges = data?.managedIdentities?.edges ?? [];

    return (
        <Box>
            {edges.length > 0 &&
                <Paper variant="outlined" sx={{ marginTop: 4, marginBottom: 4 }}>
                    <Box padding={2}>
                        <Typography gutterBottom>
                            Assign Managed Identity
                        </Typography>
                        <Typography variant="body2">
                            The managed identities assigned to this workspace will be automatically used by runs triggered against this workspace.
                        </Typography>
                        <Box display="flex" marginTop={2}>
                            <ManagedIdentityAutocomplete
                                value={selected}
                                namespacePath={workspacePath}
                                assignedManagedIdentityIDs={assignedManagedIdentityIds}
                                onSelected={onManagedIdentitySelected}
                            />
                            <Button
                                loading={assignCommitInFlight}
                                sx={{ marginLeft: 1 }}
                                variant="outlined"
                                disabled={!selected}
                                onClick={assignManagedIdentity}
                            >
                                Assign
                            </Button>
                        </Box>
                        {error && <Alert sx={{ marginTop: 2 }} severity={error.severity}>
                            {error.message}
                        </Alert>}
                    </Box>
                </Paper>}
            {edges.length === 0 && <NoResults sx={{ mt: 4 }}>
                No managed identities have been created in any parent group
            </NoResults>}

            {assignedManagedIdentities.length > 0 && <Box marginTop={2}>
                <Typography variant="h6" gutterBottom>
                    {assignedManagedIdentities.length} Assigned Managed Identit{assignedManagedIdentities.length === 1 ? 'y' : 'ies'}
                </Typography>
                <ResponsiveTable
                    ariaLabel="assigned managed identities"
                    columns={[
                        { label: 'Name' },
                        { label: 'Group' },
                        { label: 'Type' },
                        { label: '', align: 'right' },
                    ]}
                >
                    {assignedManagedIdentities.map((identity: any) => <AssignedManagedIdentityListItem
                        key={identity.id}
                        managedIdentityKey={identity}
                        onUnassign={onUnassign}
                    />)}
                </ResponsiveTable>
            </Box>}
        </Box>
    )
}

export default AssignedManagedIdentityList
