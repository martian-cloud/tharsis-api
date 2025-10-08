import { Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import ActivityEventList from '../../activity/ActivityEventList';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import { NamespaceActivityConnectionFragment_activity$key } from './__generated__/NamespaceActivityConnectionFragment_activity.graphql';
import { NamespaceActivityFragment_activity$key } from './__generated__/NamespaceActivityFragment_activity.graphql';
import { NamespaceActivityQuery } from './__generated__/NamespaceActivityQuery.graphql';

interface Props {
    fragmentRef: NamespaceActivityFragment_activity$key
}

function NamespaceActivity(props: Props) {
    const namespace = useFragment<NamespaceActivityFragment_activity$key>(
        graphql`
        fragment NamespaceActivityFragment_activity on Namespace
        {
            __typename
            fullPath
        }
      `, props.fragmentRef);

    const queryData = useLazyLoadQuery<NamespaceActivityQuery>(graphql`
        query NamespaceActivityQuery($first: Int, $last: Int, $after: String, $before: String, $namespacePath: String) {
            ...NamespaceActivityConnectionFragment_activity
        }
    `, { first: 20, namespacePath: namespace.fullPath }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<NamespaceActivityQuery, NamespaceActivityConnectionFragment_activity$key>(
        graphql`
      fragment NamespaceActivityConnectionFragment_activity on Query
      @refetchable(queryName: "NamespaceActivityPaginationQuery") {
        activityEvents(
            after: $after
            before: $before
            first: $first
            last: $last
            namespacePath: $namespacePath
            includeNested: true
            sort: CREATED_DESC
        ) @connection(key: "NamespaceActivity_activityEvents") {
            edges {
                node {
                    id
                }
            }
            ...ActivityEventListFragment_connection
        }
      }
    `, queryData);

    const eventCount = data.activityEvents?.edges?.length || 0;

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={namespace.fullPath}
                childRoutes={[
                    { title: "activity", path: 'activity' }
                ]}
            />
            {eventCount > 0 && <ActivityEventList fragmentRef={data.activityEvents} loadNext={loadNext} hasNext={hasNext} />}
            {eventCount === 0 && <Box sx={{ marginTop: 8 }} display="flex" justifyContent="center">
                <Typography variant="h6" color="textSecondary">
                    This {namespace.__typename === 'Group' ? 'group' : 'workspace'} does not have any activity events
                </Typography>
            </Box>}
        </Box>
    );
}

export default NamespaceActivity;
