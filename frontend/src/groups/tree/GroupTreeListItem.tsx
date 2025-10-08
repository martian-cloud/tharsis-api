import ExpandIcon from '@mui/icons-material/ExpandMore';
import { Avatar, Box, Paper, Stack, Typography } from '@mui/material';
import teal from '@mui/material/colors/teal';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import React, { Suspense, useState } from 'react';
import { useFragment } from "react-relay/hooks";
import Link from '../../routes/Link';
import ListSkeleton from '../../skeletons/ListSkeleton';
import NestableTreeItem from './NestableTreeItem';
import NestedGroupTreeContainer from './NestedGroupTreeContainer';
import { GroupTreeListItemFragment_group$key } from './__generated__/GroupTreeListItemFragment_group.graphql';

interface Props {
    groupKey: GroupTreeListItemFragment_group$key
    nested?: boolean
    last?: boolean
}

function GroupTreeListItem(props: Props) {
    const { nested, last } = props;
    const [showNested, setShowNested] = useState(false)
    const theme = useTheme();

    const data = useFragment<GroupTreeListItemFragment_group$key>(graphql`
        fragment GroupTreeListItemFragment_group on Group {
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
        <NestableTreeItem nested={nested} last={last}>
            <Paper
                variant="outlined"
                onClick={() => setShowNested(!showNested)}
                sx={{ cursor: 'pointer', '&:hover': { boxShadow: 1 } }}
            >
                <Box display="flex" padding={1} alignItems="center" justifyContent="space-between">
                    <Avatar sx={{ width: 32, height: 32, marginRight: 2, bgcolor: teal[200] }} variant="rounded">{data.name[0].toUpperCase()}</Avatar>
                    <Box display="flex" justifyContent="space-between" alignItems="center" flex={1}>
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
                                    variant="body1"
                                    color="textPrimary"
                                    sx={{ fontWeight: "500" }}
                                    to={`/groups/${data.fullPath}`}
                                >
                                    {nested ? data.name : data.fullPath}
                                </Link>
                                <Typography variant="body2" color="textSecondary">{data.description}</Typography>
                            </Box>
                            <Stack direction="row" spacing={1} marginRight={data.descendentGroups.totalCount === 0 ? 5 : 0}>
                                {data.descendentGroups.totalCount > 0 && <React.Fragment>
                                    <Typography variant="body2" color="textSecondary">{data.descendentGroups.totalCount} subgroup{data.descendentGroups.totalCount !== 1 ? 's' : ''}</Typography>
                                </React.Fragment>}
                                {data.workspaces.totalCount > 0 && <React.Fragment>
                                    <Typography variant="body2" color="textSecondary">{data.workspaces.totalCount} workspace{data.workspaces.totalCount !== 1 ? 's' : ''}</Typography>
                                </React.Fragment>}
                            </Stack>
                        </Box>
                        {data.descendentGroups.totalCount > 0 && <ExpandIcon color="action" sx={{ marginLeft: 1 }} transform={showNested ? 'rotate(180)' : ''} />}
                    </Box>
                </Box>
            </Paper>
            {showNested && <Box>
                <Suspense fallback={<ListSkeleton rowCount={5} />}>
                    <NestedGroupTreeContainer parentPath={data.fullPath} />
                </Suspense>
            </Box>}
        </NestableTreeItem>
    )
}

export default GroupTreeListItem
