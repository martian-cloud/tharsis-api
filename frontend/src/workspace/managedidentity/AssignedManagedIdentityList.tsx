import { LoadingButton } from '@mui/lab';
import { Alert, Box, Paper, Typography } from '@mui/material';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import React, { useState } from 'react';
import { useFragment, useMutation } from "react-relay/hooks";
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import AssignedManagedIdentityListItem from './AssignedManagedIdentityListItem';
import ManagedIdentityAutocomplete, { ManagedIdentityOption } from './ManagedIdentityAutocomplete';
import { AssignedManagedIdentityListFragment_assignedManagedIdentities$key } from './__generated__/AssignedManagedIdentityListFragment_assignedManagedIdentities.graphql';
import { AssignedManagedIdentityListMutation } from './__generated__/AssignedManagedIdentityListMutation.graphql';
import { AssignedManagedIdentityListUnassignMutation } from './__generated__/AssignedManagedIdentityListUnassignMutation.graphql';

interface Props {
    fragmentRef: AssignedManagedIdentityListFragment_assignedManagedIdentities$key
}

function AssignedManagedIdentityList(props: Props) {
    const [selected, setSelected] = useState<ManagedIdentityOption | null>(null);
    const [error, setError] = useState<MutationError | null>()
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<AssignedManagedIdentityListFragment_assignedManagedIdentities$key>(graphql`
        fragment AssignedManagedIdentityListFragment_assignedManagedIdentities on Workspace {
            id
            fullPath
            managedIdentities(includeInherited: true, first: 0) {
                totalCount
            }
            assignedManagedIdentities {
                id
                ...AssignedManagedIdentityListItemFragment_managedIdentity
            }
        }
    `, props.fragmentRef)

    const [commitAssign, assignCommitInFlight] = useMutation<AssignedManagedIdentityListMutation>(graphql`
        mutation AssignedManagedIdentityListMutation($input: AssignManagedIdentityInput!) {
            assignManagedIdentity(input: $input) {
                workspace {
                    ...AssignedManagedIdentityListFragment_assignedManagedIdentities
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
                    ...AssignedManagedIdentityListFragment_assignedManagedIdentities
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
                        workspacePath: data.fullPath
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
                    workspacePath: data.fullPath
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

    const assignedManagedIdentityIds = data.assignedManagedIdentities.reduce((accumulator, item) => {
        accumulator.add(item.id);
        return accumulator;
    }, new Set());

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[
                    { title: "managed identities", path: 'managed_identities' }
                ]}
            />
            <Typography variant="h5" gutterBottom>Assigned Managed Identities</Typography>
            {(data.managedIdentities.totalCount > 0) &&
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
                            namespacePath={data.fullPath}
                            assignedManagedIdentityIDs={assignedManagedIdentityIds}
                            onSelected={onManagedIdentitySelected}
                        />
                        <LoadingButton
                            loading={assignCommitInFlight}
                            sx={{ marginLeft: 1 }}
                            variant="outlined"
                            disabled={!selected}
                            onClick={assignManagedIdentity}
                        >
                            Assign
                        </LoadingButton>
                    </Box>
                    {error && <Alert sx={{ marginTop: 2 }} severity={error.severity}>
                        {error.message}
                    </Alert>}
                </Box>
            </Paper>}
            {data.managedIdentities.totalCount === 0 && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography variant="h6" color="textSecondary" align="center">No managed identities have been created in any parent group</Typography>
                </Box>
            </Paper>}

            {data.assignedManagedIdentities.length > 0 && <Box marginTop={2}>
                <Typography variant="h6" gutterBottom>
                    {data.assignedManagedIdentities.length} Assigned Managed Identit{data.assignedManagedIdentities.length === 1 ? 'y' : 'ies'}
                </Typography>
                <TableContainer>
                    <Table aria-label="assigned managed identities" sx={{ tableLayout: 'fixed' }}>
                        <TableHead>
                            <TableRow>
                                <TableCell>Name</TableCell>
                                <TableCell>Group</TableCell>
                                <TableCell>Type</TableCell>
                                <TableCell>Actions</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {data.assignedManagedIdentities.map((identity: any) => <AssignedManagedIdentityListItem
                                key={identity.id}
                                managedIdentityKey={identity}
                                onUnassign={onUnassign}
                            />)}
                        </TableBody>
                    </Table>
                </TableContainer>
            </Box>}
        </Box>
    )
}

export default AssignedManagedIdentityList
