import { Box, Chip, List, ListItem, ListItemText, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import Timestamp from '../../common/Timestamp';
import Link from '../../routes/Link';
import ListSkeleton from '../../skeletons/ListSkeleton';
import { ServiceAccountNamespaceMembershipsFragment$key } from './__generated__/ServiceAccountNamespaceMembershipsFragment.graphql';
import { ServiceAccountNamespaceMembershipsPaginationQuery } from './__generated__/ServiceAccountNamespaceMembershipsPaginationQuery.graphql';
import { ServiceAccountNamespaceMembershipsQuery } from './__generated__/ServiceAccountNamespaceMembershipsQuery.graphql';

const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query ServiceAccountNamespaceMembershipsQuery($id: String!, $first: Int!, $after: String) {
        node(id: $id) {
            ... on ServiceAccount {
                ...ServiceAccountNamespaceMembershipsFragment
            }
        }
    }
`;

interface Props {
    serviceAccountId: string;
}

function ServiceAccountNamespaceMemberships({ serviceAccountId }: Props) {
    const theme = useTheme();

    const queryData = useLazyLoadQuery<ServiceAccountNamespaceMembershipsQuery>(
        query,
        { id: serviceAccountId, first: INITIAL_ITEM_COUNT },
        { fetchPolicy: 'store-and-network' }
    );

    const { data, loadNext, hasNext } = usePaginationFragment<ServiceAccountNamespaceMembershipsPaginationQuery, ServiceAccountNamespaceMembershipsFragment$key>(
        graphql`
            fragment ServiceAccountNamespaceMembershipsFragment on ServiceAccount
            @refetchable(queryName: "ServiceAccountNamespaceMembershipsPaginationQuery") {
                namespaceMemberships(
                    first: $first
                    after: $after
                ) @connection(key: "ServiceAccountNamespaceMemberships_namespaceMemberships") {
                    totalCount
                    edges {
                        node {
                            id
                            metadata {
                                updatedAt
                            }
                            namespace {
                                fullPath
                            }
                            role {
                                name
                            }
                        }
                    }
                }
            }
        `,
        queryData.node
    );

    if (!data?.namespaceMemberships?.edges || data.namespaceMemberships.edges.length === 0) {
        return (
            <Paper sx={{ padding: 2 }}>
                <Typography variant="body2" color="textSecondary">
                    This service account is not a member of any namespace.
                </Typography>
            </Paper>
        );
    }

    return (
        <Box>
            <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                <Box padding={2}>
                    <Typography variant="subtitle1">
                        {data.namespaceMemberships.totalCount} namespace membership{data.namespaceMemberships.totalCount !== 1 && 's'}
                    </Typography>
                </Box>
            </Paper>
            <InfiniteScroll
                dataLength={data.namespaceMemberships.edges.length}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List disablePadding>
                    {data.namespaceMemberships.edges.filter(edge => edge?.node).map((edge: any) => (
                        <ListItem
                            key={edge.node.id}
                            sx={{
                                borderBottom: `1px solid ${theme.palette.divider}`,
                                borderLeft: `1px solid ${theme.palette.divider}`,
                                borderRight: `1px solid ${theme.palette.divider}`,
                                '&:last-child': {
                                    borderBottomLeftRadius: 4,
                                    borderBottomRightRadius: 4
                                }
                            }}
                        >
                            <ListItemText
                                primary={
                                    <Box display="flex" alignItems="center" gap={1}>
                                        <Link color="inherit" to={`/groups/${edge.node.namespace?.fullPath}/-/members`}>
                                            <Typography fontWeight={500}>{edge.node.namespace?.fullPath}</Typography>
                                        </Link>
                                        <Chip size="small" variant="outlined" label={edge.node.role.name} />
                                    </Box>
                                }
                            />
                            <Timestamp variant="body2" color="textSecondary" timestamp={edge.node.metadata.updatedAt} />
                        </ListItem>
                    ))}
                </List>
            </InfiniteScroll>
        </Box>
    );
}

export default ServiceAccountNamespaceMemberships;
