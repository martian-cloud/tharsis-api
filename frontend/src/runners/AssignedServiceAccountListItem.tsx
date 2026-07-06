import DeleteIcon from '@mui/icons-material/CloseOutlined';
import { Avatar, Box, Button, ListItem, ListItemText, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import Timestamp from '../common/Timestamp';
import Link from '../routes/Link';
import { AssignedServiceAccountListItemFragment_assignedServiceAccount$key } from './__generated__/AssignedServiceAccountListItemFragment_assignedServiceAccount.graphql';

interface Props {
    fragmentRef: AssignedServiceAccountListItemFragment_assignedServiceAccount$key
    onDelete: (resourcePath: string) => void
}

function AssignedServiceAccountListItem({ fragmentRef, onDelete }: Props) {
    const theme = useTheme();

    const data = useFragment(graphql`
        fragment AssignedServiceAccountListItemFragment_assignedServiceAccount on ServiceAccount {
            id
            name
            resourcePath
            groupPath
            description
            metadata {
                updatedAt
            }
        }
    `, fragmentRef);

    const route = `/groups/${data.groupPath}/-/service_accounts/${data.id}`;

    return (
        <ListItem
            dense
            sx={{
                paddingY: 1.5,
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                flexDirection: { xs: 'column', sm: 'row' },
                alignItems: { xs: 'stretch', sm: 'center' },
                gap: { xs: 1, sm: 0 },
                '&:last-child': {
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }
            }}>
            <Box display="flex" alignItems="center" sx={{ minWidth: 0, flexGrow: 1, width: { xs: '100%', sm: 'auto' } }}>
                <Avatar variant="rounded" sx={{ width: 32, height: 32, bgcolor: 'avatar.serviceAccount', marginRight: 2, flexShrink: 0 }}>
                    {data.name[0].toUpperCase()}
                </Avatar>
                <ListItemText
                    sx={{ minWidth: 0, my: 0 }}
                    primary={
                        <Link
                            to={route}
                            color='inherit'
                            sx={{ fontSize: '16px', textDecoration: 'none', wordBreak: 'break-word' }}
                        >
                            {data.name}
                        </Link>}
                    secondary={data.description} />
            </Box>
            <Box display="flex" alignItems="center" gap={1} sx={{ width: { xs: '100%', sm: 'auto' }, justifyContent: { xs: 'space-between', sm: 'flex-end' }, flexShrink: 0 }}>
                <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.updatedAt as string} />
                <Button
                    onClick={() => onDelete(data.resourcePath)}
                    sx={{ minWidth: 40, padding: '2px' }}
                    size="small"
                    color="info"
                    variant="outlined">
                    <DeleteIcon />
                </Button>
            </Box>
        </ListItem>
    );
}

export default AssignedServiceAccountListItem
