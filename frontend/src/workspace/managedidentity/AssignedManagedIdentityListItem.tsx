import DeleteIcon from '@mui/icons-material/Close';
import IconButton from '@mui/material/IconButton';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import ManagedIdentityTypeChip from '../../groups/managedidentity/ManagedIdentityTypeChip';
import { ResponsiveRow } from '../../common/ResponsiveTable';
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
        <ResponsiveRow cells={[
            {
                primary: true, content: <Link
                    to={`/groups/${groupPath}/-/managed_identities/${data.id}`}
                    color="inherit"
                    variant="body1"
                    sx={{ wordBreak: 'break-word' }}
                >
                    {data.name}
                </Link>
            },
            {
                label: 'Group', content: <Link
                    to={`/groups/${groupPath}`}
                    color="inherit"
                    variant="body1"
                    sx={{ wordBreak: 'break-word' }}
                >
                    {groupPath}
                </Link>
            },
            { label: 'Type', content: <ManagedIdentityTypeChip type={data.type} /> },
            {
                align: 'right', content: <IconButton onClick={() => onUnassign(data.id)}>
                    <DeleteIcon />
                </IconButton>
            },
        ]} />
    )
}

export default AssignedManagedIdentityListItem
