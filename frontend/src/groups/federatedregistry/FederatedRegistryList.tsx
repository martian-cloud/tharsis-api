import { Box, Button, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, useFragment, useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import { FederatedRegistryListFragment_group$key } from './__generated__/FederatedRegistryListFragment_group.graphql';
import { FederatedRegistryListFragment_federatedRegistries$key } from './__generated__/FederatedRegistryListFragment_federatedRegistries.graphql';
import { FederatedRegistryListPaginationQuery } from './__generated__/FederatedRegistryListPaginationQuery.graphql';
import { FederatedRegistryListQuery } from './__generated__/FederatedRegistryListQuery.graphql';
import FederatedRegistryListItem from './FederatedRegistryListItem';

const DESCRIPTION = 'Federated registries enable access to Terraform modules and providers from external Tharsis registries';
const INITIAL_ITEM_COUNT = 100;

const NewButton =
    <Button
        component={RouterLink}
        variant="outlined"
        to="new"
        sx={{
            textAlign: 'center',
            width: { xs: '100%', sm: 'auto' }
        }}
    >
        New Federated Registry
    </Button>;

const query = graphql`
    query FederatedRegistryListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!) {
        node(id: $groupId) {
            ...on Group {
                ...FederatedRegistryListFragment_federatedRegistries
            }
        }
    }
`;

export function GetConnections(groupId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        groupId,
        'FederatedRegistryList_federatedRegistries',
        { sort: 'UPDATED_AT_DESC' }
    );
    return [connectionId];
}

interface Props {
    fragmentRef: FederatedRegistryListFragment_group$key;
}

function FederatedRegistryList({ fragmentRef }: Props) {
    const theme = useTheme();

    const group = useFragment<FederatedRegistryListFragment_group$key>(
        graphql`
        fragment FederatedRegistryListFragment_group on Group
        {
            id
            fullPath
        }
    `, fragmentRef);

    const queryData = useLazyLoadQuery<FederatedRegistryListQuery>(query, { first: INITIAL_ITEM_COUNT, groupId: group.id }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<FederatedRegistryListPaginationQuery, FederatedRegistryListFragment_federatedRegistries$key>(
        graphql`
      fragment FederatedRegistryListFragment_federatedRegistries on Group
      @refetchable(queryName: "FederatedRegistryListPaginationQuery") {
            federatedRegistries(
                after: $after
                before: $before
                first: $first
                last: $last
                sort: UPDATED_AT_DESC
            ) @connection(key: "FederatedRegistryList_federatedRegistries") {
                totalCount
                edges {
                    node {
                        id
                        ...FederatedRegistryListItemFragment_federatedRegistry
                    }
                }
            }
      }
    `, queryData.node);

    const federatedRegistries = data?.federatedRegistries?.edges;
    const totalCount = data?.federatedRegistries?.totalCount ?? 0;
    const hasRegistries = federatedRegistries && federatedRegistries.length > 0;

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "federated registries", path: 'federated_registries' }
                ]}
            />
            <Box>
                {hasRegistries && <Box>
                    <Box sx={{
                        display: 'flex',
                        flexDirection: 'row',
                        justifyContent: 'space-between',
                        [theme.breakpoints.down('md')]: {
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            '& > *': { mb: 2 },
                        }
                    }}>
                        <Box>
                            <Typography variant="h5" gutterBottom>Federated Registries</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                        <Box>{NewButton}</Box>
                    </Box>
                    <Paper sx={{
                        mt: 2,
                        borderBottomLeftRadius: 0,
                        borderBottomRightRadius: 0,
                        border: `1px solid ${theme.palette.divider}`
                    }}>
                        <Box p={2} display="flex" alignItems="center" justifyContent="space-between">
                            <Typography variant="subtitle1">
                                {totalCount} federated registr{totalCount === 1 ? 'y' : 'ies'}
                            </Typography>
                        </Box>
                    </Paper>
                    <InfiniteScroll
                        dataLength={federatedRegistries?.length ?? 0}
                        next={() => loadNext(20)}
                        hasMore={hasNext}
                        loader={<ListSkeleton rowCount={3} />}
                    >
                        <List disablePadding>
                            {federatedRegistries?.map((edge) => edge?.node && <FederatedRegistryListItem
                                key={edge.node.id}
                                fragmentRef={edge.node}
                            />)}
                        </List>
                    </InfiniteScroll>
                </Box>}

                {!hasRegistries && <Box sx={{ mt: 4 }} display="flex" justifyContent="center">
                    <Box display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ p: 4, maxWidth: 600 }}>
                        <Typography variant="h6">Get started with federated registries</Typography>
                        <Typography color="textSecondary" align="center" sx={{ mb: 2 }}>
                            {DESCRIPTION}
                        </Typography>
                        <Box>
                            {NewButton}
                        </Box>
                    </Box>
                </Box>
                }
            </Box>
        </Box>
    );
}

export default FederatedRegistryList;
