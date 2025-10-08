import LoadMoreIcon from '@mui/icons-material/DoubleArrowOutlined';
import { Box, CircularProgress, Paper, Typography } from '@mui/material';
import List from '@mui/material/List';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { LoadMoreFn, useFragment } from "react-relay/hooks";
import GroupTreeListItem from './GroupTreeListItem';
import NestableTreeItem from './NestableTreeItem';
import { GroupTreeFragment_connection$key } from './__generated__/GroupTreeFragment_connection.graphql';
import { NestedGroupsListPaginationQuery } from './__generated__/NestedGroupsListPaginationQuery.graphql';

interface Props {
    connectionKey: GroupTreeFragment_connection$key
    loadNext: LoadMoreFn<NestedGroupsListPaginationQuery>
    hasNext: boolean
    isLoadingNext: boolean
    isRefreshing?: boolean
    nested?: boolean
}

function GroupTree(props: Props) {
    const { connectionKey, nested, loadNext, hasNext, isLoadingNext, isRefreshing } = props;

    const data = useFragment<GroupTreeFragment_connection$key>(graphql`
        fragment GroupTreeFragment_connection on GroupConnection {
            totalCount
            edges {
                node {
                    id
                    ...GroupTreeListItemFragment_group
                }
            }
        }
    `, connectionKey);

    return (
        <Box>
            <List disablePadding sx={isRefreshing ? { opacity: 0.5 } : null}>
                {data.edges?.map((edge: any, index: number) => <GroupTreeListItem key={edge.node.id} groupKey={edge.node} nested={nested} last={index === (data.totalCount - 1)} />)}
                {hasNext && <NestableTreeItem nested={nested} last={true}>
                    <Paper
                        variant="outlined"
                        onClick={() => loadNext(100)}
                        sx={{ cursor: 'pointer', '&:hover': { boxShadow: 1 }, display: 'flex', alignItems: 'center', padding: 1 }}
                    >
                        <Box width={32} display="flex" alignItems="center" justifyContent="center" marginRight={2}>
                            {!isLoadingNext && <LoadMoreIcon color="action" />}
                            {isLoadingNext && <CircularProgress size={24} />}
                        </Box>
                        <Typography color="textSecondary" sx={{ fontWeight: "500" }}>View More</Typography>
                    </Paper>
                </NestableTreeItem>}
            </List>
        </Box>
    );
}

export default GroupTree;
