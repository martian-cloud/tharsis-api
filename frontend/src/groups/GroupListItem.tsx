import { Avatar, Box, Stack, Typography } from '@mui/material';
import teal from '@mui/material/colors/teal';
import Link from '@mui/material/Link';
import ListItem from '@mui/material/ListItem';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from "react-relay/hooks";
import { Link as LinkRouter } from 'react-router-dom';
import { GroupListItemFragment_group$key } from './__generated__/GroupListItemFragment_group.graphql';

interface Props {
    groupKey: GroupListItemFragment_group$key
    last?: boolean
}

function GroupListItem(props: Props) {
    const { last } = props;
    const theme = useTheme();

    const data = useFragment<GroupListItemFragment_group$key>(graphql`
        fragment GroupListItemFragment_group on Group {
            metadata {
                updatedAt
            }
            id
            name
            description
            fullPath
            descendentGroups(first: 0) {
                totalCount
            }
            workspaces(first: 0) {
                totalCount
            }
        }
    `, props.groupKey)

    return (
        <ListItem button disablePadding divider={!last} component={LinkRouter} to={`/groups/${data.fullPath}`}>
            <Box flex={1} display="flex" padding={1} alignItems="center">
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
                    <Box>
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
                    </Box>
                    <Stack direction="row" spacing={1}>
                        {data.descendentGroups.totalCount > 0 && <React.Fragment>
                            <Typography variant="body2" color="textSecondary">{data.descendentGroups.totalCount} subgroup{data.descendentGroups.totalCount !== 1 ? 's' : ''}</Typography>
                        </React.Fragment>}
                        {data.workspaces.totalCount > 0 && <React.Fragment>
                            <Typography variant="body2" color="textSecondary">{data.workspaces.totalCount} workspace{data.workspaces.totalCount !== 1 ? 's' : ''}</Typography>
                        </React.Fragment>}
                    </Stack>
                </Box>
            </Box>
        </ListItem>
    )
}

export default GroupListItem
