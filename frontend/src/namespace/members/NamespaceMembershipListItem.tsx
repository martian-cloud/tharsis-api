import DeleteIcon from '@mui/icons-material/CloseOutlined';
import EditIcon from '@mui/icons-material/EditOutlined';
import LoadingButton from '@mui/lab/LoadingButton';
import { Avatar, Box, Button, Stack } from '@mui/material';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import teal from '@mui/material/colors/teal';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import React, { useState } from 'react';
import { useFragment, useMutation } from "react-relay/hooks";
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import Link from '../../routes/Link';
import RoleAutocomplete from './RoleAutocomplete';
import { NamespaceMembershipListItemFragment_membership$key } from './__generated__/NamespaceMembershipListItemFragment_membership.graphql';
import { NamespaceMembershipListItemUpdateNamespaceMembershipMutation } from './__generated__/NamespaceMembershipListItemUpdateNamespaceMembershipMutation.graphql';

interface Props {
    fragmentRef: NamespaceMembershipListItemFragment_membership$key
    namespacePath: string
    onDelete: (membership: any) => void
}

function NamespaceMembershipListItem(props: Props) {
    const { fragmentRef, namespacePath, onDelete } = props;
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<NamespaceMembershipListItemFragment_membership$key>(graphql`
        fragment NamespaceMembershipListItemFragment_membership on NamespaceMembership {
            metadata {
                createdAt
                updatedAt
                trn
            }
            id
            role {
                name
            }
            resourcePath
            member {
                __typename
                ...on User {
                    id
                    username
                    email
                }
                ...on Team {
                    id
                    name
                }
                ...on ServiceAccount {
                    id
                    name
                    resourcePath
                }
            }
        }
    `, fragmentRef);

    const [commitUpdateNamespaceMembership, updateInFlight] = useMutation<NamespaceMembershipListItemUpdateNamespaceMembershipMutation>(graphql`
        mutation NamespaceMembershipListItemUpdateNamespaceMembershipMutation($input: UpdateNamespaceMembershipInput!) {
            updateNamespaceMembership(input: $input) {
                namespace {
                    memberships {
                        ...NamespaceMembershipListItemFragment_membership
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [editMode, setEditMode] = useState(false);
    const [role, setRole] = useState(data.role?.name);

    const onSave = () => {
        commitUpdateNamespaceMembership({
            variables: {
                input: {
                    id: data.id,
                    role
                },
            },
            onCompleted: data => {
                if (data.updateNamespaceMembership.problems.length) {
                    enqueueSnackbar(data.updateNamespaceMembership.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                } else {
                    setEditMode(false);
                }
            },
            onError: error => {
                enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
            }
        });
    };

    const type = data.member?.__typename;
    const membershipNamespacePath = data.resourcePath.split("/").slice(0, -1).join("/");

    return (
        <TableRow>
            <TableCell sx={{ fontWeight: 'bold' }}>
                <Stack direction="row" alignItems="center" spacing={1}>
                    {type === 'User' && <React.Fragment>
                        <Gravatar width={24} height={24} sx={{ marginRight: 1 }} email={data.member?.email ?? ''} />
                        <Box>{data.member?.username}</Box>
                    </React.Fragment>}
                    {type === 'Team' && <React.Fragment>
                        <Avatar variant="rounded" sx={{ width: 24, height: 24, bgcolor: teal[200], fontSize: 14, marginRight: 1 }}>
                            {(data.member?.name ?? '')[0].toUpperCase()}
                        </Avatar>
                        <Box>{data.member?.name}</Box>
                    </React.Fragment>}
                    {type === 'ServiceAccount' && <React.Fragment>
                        <Avatar variant="rounded" sx={{ width: 24, height: 24, bgcolor: teal[200], fontSize: 14, marginRight: 1 }}>
                            {data.member?.name[0].toUpperCase()}
                        </Avatar>
                        <Box>
                            <Link color="inherit" to={`/groups/${data.member?.resourcePath.split("/").slice(0, -1).join("/")}/-/service_accounts/${data.member?.id}`}>
                                {data.member?.resourcePath}
                            </Link>
                        </Box>
                    </React.Fragment>}
                </Stack>
            </TableCell>
            <TableCell>
                {data.member?.__typename}
            </TableCell>
            <TableCell>
                {editMode && <RoleAutocomplete size="small" onSelected={role => role && setRole(role.name)} />}
                {!editMode && <React.Fragment>{data.role.name}</React.Fragment>}
            </TableCell>
            <TableCell>
                <Timestamp timestamp={data.metadata.updatedAt} />
            </TableCell>
            <TableCell>
                {membershipNamespacePath === namespacePath ? 'Direct Member' : <Link
                    to={`/groups/${membershipNamespacePath}/-/members`}
                    color="inherit"
                    variant="body1"
                >
                    {membershipNamespacePath}
                </Link>}
            </TableCell>
            <TableCell>
                {editMode && <Stack direction="row" spacing={1}>
                    <LoadingButton
                        loading={updateInFlight}
                        onClick={onSave}
                        sx={{ minWidth: 40, padding: '2px' }}
                        size="small"
                        color="secondary"
                        variant="outlined">
                        Save
                    </LoadingButton>
                    <Button
                        onClick={() => setEditMode(false)}
                        sx={{ minWidth: 40, padding: '2px' }}
                        size="small"
                        color="info"
                        variant="outlined">
                        Cancel
                    </Button>
                </Stack>}
                {!editMode && namespacePath === membershipNamespacePath && <Stack direction="row" spacing={1}>
                    <TRNButton trn={data.metadata.trn} />
                    <Button
                        onClick={() => setEditMode(true)}
                        sx={{ minWidth: 40, padding: '2px' }}
                        size="small"
                        color="info"
                        variant="outlined">
                        <EditIcon />
                    </Button>
                    <Button
                        onClick={() => onDelete(data)}
                        sx={{ minWidth: 40, padding: '2px' }}
                        size="small"
                        color="info"
                        variant="outlined">
                        <DeleteIcon />
                    </Button>
                </Stack>}
            </TableCell>
        </TableRow>
    );
}

export default NamespaceMembershipListItem
