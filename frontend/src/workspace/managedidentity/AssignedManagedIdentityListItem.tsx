import DeleteIcon from '@mui/icons-material/Close';
import IconButton from '@mui/material/IconButton';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from "react-relay/hooks";
import ManagedIdentityTypeChip from '../../groups/managedidentity/ManagedIdentityTypeChip';
import Link from '../../routes/Link';
import { AssignedManagedIdentityListItemFragment_managedIdentity$key } from './__generated__/AssignedManagedIdentityListItemFragment_managedIdentity.graphql';

interface Props {
    managedIdentityKey: AssignedManagedIdentityListItemFragment_managedIdentity$key
    onUnassign: (managedIdentityId: string) => void
}

function AssignedManagedIdentityListItem(props: Props) {
    const { onUnassign } = props;

    const data = useFragment<AssignedManagedIdentityListItemFragment_managedIdentity$key>(graphql`
        fragment AssignedManagedIdentityListItemFragment_managedIdentity on ManagedIdentity {
            metadata {
                updatedAt
            }
            id
            name
            description
            type
            resourcePath
        }
    `, props.managedIdentityKey)

    const groupPath = data.resourcePath.split("/").slice(0, -1).join("/");

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <TableCell>
                <Link
                    to={`/groups/${groupPath}/-/managed_identities/${data.id}`}
                    color="inherit"
                    variant="body1"
                >
                    {data.name}
                </Link>
            </TableCell>

            <TableCell>
                <Link
                    to={`/groups/${groupPath}`}
                    color="inherit"
                    variant="body1"
                >
                    {groupPath}
                </Link>
            </TableCell>
            <TableCell>
                <ManagedIdentityTypeChip type={data.type} />
            </TableCell>
            <TableCell>
                <IconButton onClick={() => onUnassign(data.id)}>
                    <DeleteIcon />
                </IconButton>
            </TableCell>
        </TableRow >
    )
}

export default AssignedManagedIdentityListItem
