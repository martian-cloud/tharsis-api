import { Avatar, Box, Stack, Typography } from '@mui/material';
import Link from '@mui/material/Link';
import ListItemButton from '@mui/material/ListItemButton';
import { teal } from '@mui/material/colors';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { Link as LinkRouter } from 'react-router-dom';
import Timestamp from '../common/Timestamp';
import LabelList from '../workspace/labels/LabelList';
import { WorkspaceListItemFragment_workspace$key } from './__generated__/WorkspaceListItemFragment_workspace.graphql';

interface Props {
    workspaceKey: WorkspaceListItemFragment_workspace$key
    last?: boolean
}

function WorkspaceListItem(props: Props) {
    const { last } = props;
    const theme = useTheme();

    const data = useFragment<WorkspaceListItemFragment_workspace$key>(graphql`
        fragment WorkspaceListItemFragment_workspace on Workspace {
            metadata {
                updatedAt
            }
            id
            name
            description
            fullPath
            labels {
                key
                value
            }
        }
    `, props.workspaceKey)

    return (
        <ListItemButton
            divider={!last}
            component={LinkRouter}
            to={`/groups/${data.fullPath}`}>
            <Avatar sx={{ width: 32, height: 32, marginRight: 2, bgcolor: teal[200] }} variant="rounded">{data.name[0].toUpperCase()}</Avatar>
            <Box sx={{
                display: 'flex',
                flexDirection: 'column',
                [theme.breakpoints.down('md')]: {
                    '& > *:nth-of-type(2)': {
                        marginTop: 0.5
                    }
                },
                [theme.breakpoints.up('md')]: {
                    flexGrow: 1,
                    flexDirection: 'row',
                    alignItems: 'center',
                    justifyContent: 'space-between'
                }
            }}>
                <Box sx={{ flex: 1 }}>
                    <Link
                        component="div"
                        underline="hover"
                        variant="body1"
                        color="textPrimary"
                        sx={{ fontWeight: "500" }}
                    >
                        {data.name}
                    </Link>
                    <Typography variant="body2" color="textSecondary">{data.description}</Typography>
                    {data.labels.length > 0 && (
                        <Box sx={{ mt: 0.5 }}>
                            <LabelList
                                labels={data.labels as any}
                                size="xs"
                                maxVisible={3}
                            />
                        </Box>
                    )}
                </Box>
                <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <Timestamp variant="body2" color="textSecondary" timestamp={data.metadata.updatedAt} />
                </Stack>
            </Box>
        </ListItemButton>
    );
}

export default WorkspaceListItem
