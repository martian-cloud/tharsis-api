import { Box, Button, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React, { Suspense } from 'react';
import { PreloadedQuery, useFragment, usePreloadedQuery } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import ListSkeleton from '../../skeletons/ListSkeleton';
import NamespaceList from '../NamespaceList';
import { GroupTreeContainerFragment_me$key } from './__generated__/GroupTreeContainerFragment_me.graphql';
import { GroupTreeContainerQuery } from './__generated__/GroupTreeContainerQuery.graphql';
import { PageLayoutProvider } from '../../layout/PageLayoutContext';

const query = graphql`
    query GroupTreeContainerQuery {
      ...GroupTreeContainerFragment_me
    }
`;

interface Props {
    queryRef: PreloadedQuery<GroupTreeContainerQuery>;
}

function GroupTreeContainer(props: Props) {
    const queryData = usePreloadedQuery<GroupTreeContainerQuery>(query, props.queryRef);

    const userData = useFragment<GroupTreeContainerFragment_me$key>(
        graphql`
        fragment GroupTreeContainerFragment_me on Query
        {
            me {
                ...on User {
                    adminModeEnabled
                }
            }
        }
      `, queryData);

    const isAdmin = userData.me?.adminModeEnabled;

    const emptyState = (
        <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
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
        </Box>
    );

    const header = (
        <Box display="flex" justifyContent="space-between">
            <Typography variant="h5" mb={2}>Groups</Typography>
            {isAdmin && <Box>
                <Button component={RouterLink} variant="outlined" color="primary" to={`/groups/-/new`}>New Group</Button>
            </Box>}
        </Box>
    );

    return (
        <PageLayoutProvider>
            <Suspense fallback={<ListSkeleton rowCount={5} />}>
                <NamespaceList header={header} emptyState={emptyState} />
            </Suspense>
        </PageLayoutProvider>
    );
}

export default GroupTreeContainer;
