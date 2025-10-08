import DeleteIcon from '@mui/icons-material/CloseOutlined';
import { Avatar, Button, ListItem, ListItemText, Typography, useTheme } from '@mui/material';
import purple from '@mui/material/colors/purple';
import graphql from 'babel-plugin-relay/macro';
import moment from 'moment';
import { useFragment } from 'react-relay/hooks';
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
                paddingY: data.description ? 0 : 1.5,
                borderBottom: `1px solid ${theme.palette.divider}`,
                borderLeft: `1px solid ${theme.palette.divider}`,
                borderRight: `1px solid ${theme.palette.divider}`,
                '&:last-child': {
                    borderBottomLeftRadius: 4,
                    borderBottomRightRadius: 4
                }
            }}>
            <Avatar variant="rounded" sx={{ width: 32, height: 32, bgcolor: purple[300], marginRight: 2 }}>
                {data.name[0].toUpperCase()}
            </Avatar>
            <ListItemText
                primary={
                    <Link
                        to={route}
                        color='inherit'
                        sx={{ fontSize: '16px', textDecoration: 'none' }}
                    >
                        {data.name}
                    </Link>}
                secondary={data.description} />
            <Typography variant="body2" color="textSecondary">
                {moment(data.metadata.updatedAt as moment.MomentInput).fromNow()}
            </Typography>
            <Button
                onClick={() => onDelete(data.resourcePath)}
                sx={{ ml: 2, minWidth: 40, padding: '2px' }}
                size="small"
                color="info"
                variant="outlined">
                <DeleteIcon />
            </Button>
        </ListItem>
    );
}

export default AssignedServiceAccountListItem
