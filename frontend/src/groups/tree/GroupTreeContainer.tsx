import { Box, Button, Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useMemo, useState } from 'react';
import { ConnectionHandler, PreloadedQuery, fetchQuery, useFragment, usePaginationFragment, usePreloadedQuery, useRelayEnvironment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import SearchInput from '../../common/SearchInput';
import GroupTree from './GroupTree';
import { GroupTreeContainerFragment_groups$key } from './__generated__/GroupTreeContainerFragment_groups.graphql';
import { GroupTreeContainerFragment_me$key } from './__generated__/GroupTreeContainerFragment_me.graphql';
import { GroupTreeContainerQuery } from './__generated__/GroupTreeContainerQuery.graphql';
import { GroupsPaginationQuery } from './__generated__/GroupsPaginationQuery.graphql';

export const INITIAL_ITEM_COUNT = 100;
export const DEFAULT_SORT = 'FULL_PATH_ASC';

const query = graphql`
    query GroupTreeContainerQuery($first: Int, $last: Int, $after: String, $before: String, $search: String, $parentPath: String, $sort: GroupSort) {
      ...GroupTreeContainerFragment_groups
      ...GroupTreeContainerFragment_me
    }
`;

interface Props {
    queryRef: PreloadedQuery<GroupTreeContainerQuery>;
}

export function GetConnections() {
    const connectionId = ConnectionHandler.getConnectionID(
        "root",
        "GroupTreeContainer_groups",
        { sort: DEFAULT_SORT, parentPath: '' }
    );
    return [connectionId]
}

function GroupTreeContainer(props: Props) {
    const queryData = usePreloadedQuery<GroupTreeContainerQuery>(query, props.queryRef);

    const userData = useFragment<GroupTreeContainerFragment_me$key>(
        graphql`
        fragment GroupTreeContainerFragment_me on Query
        {
            me {
                ...on User {
                    admin
                }
            }
        }
      `, queryData);

    const { data, loadNext, hasNext, isLoadingNext, refetch } = usePaginationFragment<GroupsPaginationQuery, GroupTreeContainerFragment_groups$key>(
        graphql`
        fragment GroupTreeContainerFragment_groups on Query
        @refetchable(queryName: "GroupsPaginationQuery") {
            groups(
                after: $after
                before: $before
                first: $first
                last: $last
                search: $search
                parentPath: $parentPath
                sort: $sort
            ) @connection(key: "GroupTreeContainer_groups") {
                totalCount
                edges {
                    node {
                        id
                    }
                }
                ...GroupTreeFragment_connection
            }
        }
    `, queryData);

    const [search, setSearch] = useState<string | undefined>('');
    const [isRefreshing, setIsRefreshing] = useState(false);

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input?: string) => {
                    setIsRefreshing(true);

                    const normalizedInput = input?.trim();
                    const sort = input === '' ? DEFAULT_SORT : 'GROUP_LEVEL_ASC';

                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, search: normalizedInput, sort })
                        .subscribe({
                            complete: () => {
                                setIsRefreshing(false);
                                setSearch(input);

                                // *After* the query has been fetched, we call
                                // refetch again to re-render with the updated data.
                                // At this point the data for the query should
                                // be cached, so we use the 'store-only'
                                // fetchPolicy to avoid suspending.
                                refetch({ first: INITIAL_ITEM_COUNT, search: input === '' ? null : normalizedInput, parentPath: input === '' ? '' : null, sort }, { fetchPolicy: 'store-only' });
                            },
                            error: () => {
                                setIsRefreshing(false);
                            }
                        });
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment, refetch],
    );

    const isAdmin = userData.me?.admin;

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase());
    };

    const onKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        // Only handle enter key type
        if (event.which === 13) {
            fetch.flush();
        }
    };

    return (
        <Box maxWidth={1200} margin="auto" padding={2}>
            {(search !== '' || (data.groups.edges && data.groups.edges.length > 0)) && <Box>
                <Box display="flex" justifyContent="space-between">
                    <Box marginBottom={2}>
                        <Typography variant="h5">Groups</Typography>
                    </Box>
                    {isAdmin && <Box>
                        <Button component={RouterLink} variant="outlined" color="primary" to={`/groups/-/new`}>New Group</Button>
                    </Box>}
                </Box>
                <Box sx={{ mb: 1 }}>
                    <SearchInput
                        fullWidth
                        placeholder="search for groups"
                        onChange={onSearchChange}
                        onKeyPress={onKeyPress}
                    />
                </Box>
                {(!data.groups.edges || data.groups.edges?.length === 0) && search !== '' && <Paper
                    variant="outlined"
                    sx={{ p: 4, mt: 2, textAlign: "center" }}>
                    <Typography color="textSecondary">No groups matching search <strong>{search}</strong>
                    </Typography>
                </Paper>}
                <GroupTree connectionKey={data.groups} loadNext={loadNext} hasNext={hasNext} isLoadingNext={isLoadingNext} isRefreshing={isRefreshing} />
            </Box>}
            {((!data.groups.edges || data.groups.edges.length === 0) && search === '') && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    {!isAdmin && <React.Fragment>
                        <Typography variant="h6">You're not a member of any groups</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            To get started you can request access to an existing group or contact the system administrator to create a new group
                        </Typography>
                    </React.Fragment>}
                    {isAdmin && <React.Fragment>
                        <Typography variant="h6">Get started by creating a new group</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            Groups are containers that hold configuration information to help organize workspaces in a hierarchical manner
                        </Typography>
                        <Button component={RouterLink} variant="outlined" color="primary" to={`/groups/-/new`}>New Group</Button>
                    </React.Fragment>}
                </Box>
            </Box>}
        </Box>
    );
}

export default GroupTreeContainer;
