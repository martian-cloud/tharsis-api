import DeleteIcon from '@mui/icons-material/CloseOutlined';
import EditIcon from '@mui/icons-material/EditOutlined';
import { Avatar, Box, Button, Stack, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import React, { useState } from 'react';
import { useFragment, useMutation } from "react-relay/hooks";
import Gravatar from '../../common/Gravatar';
import { ResponsiveRow } from '../../common/ResponsiveTable';
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

    const name = (
        <Stack direction="row" alignItems="center" spacing={1} sx={{ fontWeight: 'bold', minWidth: 0 }}>
            {type === 'User' && <React.Fragment>
                <Gravatar width={24} height={24} sx={{ marginRight: 1 }} email={data.member?.email ?? ''} />
                <Box sx={{ wordBreak: 'break-word' }}>{data.member?.username}</Box>
            </React.Fragment>}
            {type === 'Team' && <React.Fragment>
                <Avatar variant="rounded" sx={{ width: 24, height: 24, bgcolor: 'avatar.default', fontSize: 14, marginRight: 1 }}>
                    {(data.member?.name ?? '')[0].toUpperCase()}
                </Avatar>
                <Box sx={{ wordBreak: 'break-word' }}>
                    <Link color="inherit" to={`/teams/${encodeURIComponent(data.member?.name ?? '')}`}>
                        {data.member?.name}
                    </Link>
                </Box>
            </React.Fragment>}
            {type === 'ServiceAccount' && <React.Fragment>
                <Avatar variant="rounded" sx={{ width: 24, height: 24, bgcolor: 'avatar.default', fontSize: 14, marginRight: 1 }}>
                    {data.member?.name[0].toUpperCase()}
                </Avatar>
                <Box sx={{ wordBreak: 'break-word' }}>
                    <Link color="inherit" to={`/groups/${data.member?.resourcePath.split("/").slice(0, -1).join("/")}/-/service_accounts/${data.member?.id}`}>
                        {data.member?.resourcePath}
                    </Link>
                </Box>
            </React.Fragment>}
        </Stack>
    );

    const roleContent = editMode
        ? <RoleAutocomplete size="small" onSelected={role => role && setRole(role.name)} />
        : <Typography variant="body2">{data.role?.name}</Typography>;

    const source = membershipNamespacePath === namespacePath
        ? <Typography variant="body2" color="textSecondary">Direct Member</Typography>
        : <Link to={`/groups/${membershipNamespacePath}/-/members`} color="inherit" variant="body2">{membershipNamespacePath}</Link>;

    const actions = editMode ? <Stack direction="row" spacing={1} justifyContent="flex-end">
        <Button
            loading={updateInFlight}
            onClick={onSave}
            size="small"
            color="primary"
            variant="outlined">
            Save
        </Button>
        <Button
            onClick={() => setEditMode(false)}
            size="small"
            color="inherit"
            variant="outlined">
            Cancel
        </Button>
    </Stack> : (namespacePath === membershipNamespacePath ? <Stack direction="row" spacing={1} justifyContent="flex-end">
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
    </Stack> : null);

    return (
        <ResponsiveRow cells={[
            { primary: true, content: name },
            { label: 'Type', content: <Typography variant="body2">{data.member?.__typename}</Typography> },
            { label: 'Role', content: roleContent },
            { label: 'Last Updated', content: <Timestamp variant="body2" timestamp={data.metadata.updatedAt} /> },
            { label: 'Source', content: source },
            { align: 'right', content: actions },
        ]} />
    );
}

export default NamespaceMembershipListItem
