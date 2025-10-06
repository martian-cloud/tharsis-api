import { Box, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useFragment, useLazyLoadQuery, usePaginationFragment } from "react-relay/hooks";
import ListSkeleton from '../skeletons/ListSkeleton';
import TerraformProviderVersionListItem from './TerraformProviderVersionListItem';
import { TerraformProviderVersionListFragment_provider$key } from './__generated__/TerraformProviderVersionListFragment_provider.graphql';
import { TerraformProviderVersionListFragment_versions$key } from './__generated__/TerraformProviderVersionListFragment_versions.graphql';
import { TerraformProviderVersionListPaginationQuery } from './__generated__/TerraformProviderVersionListPaginationQuery.graphql';
import { TerraformProviderVersionListQuery } from './__generated__/TerraformProviderVersionListQuery.graphql';

const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query TerraformProviderVersionListQuery($first: Int, $last: Int, $after: String, $before: String, $providerId: String!) {
        ...TerraformProviderVersionListFragment_versions
    }
`;

interface Props {
    fragmentRef: TerraformProviderVersionListFragment_provider$key
}

function TerraformProviderVersionList(props: Props) {
    const theme = useTheme();
    const provider = useFragment<TerraformProviderVersionListFragment_provider$key>(
        graphql`
        fragment TerraformProviderVersionListFragment_provider on TerraformProvider
        {
          id
        }
    `, props.fragmentRef)

    const queryData = useLazyLoadQuery<TerraformProviderVersionListQuery>(query, { first: INITIAL_ITEM_COUNT, providerId: provider.id }, { fetchPolicy: 'store-and-network' })

    const { data, loadNext, hasNext } = usePaginationFragment<TerraformProviderVersionListPaginationQuery, TerraformProviderVersionListFragment_versions$key>(
        graphql`
      fragment TerraformProviderVersionListFragment_versions on Query
      @refetchable(queryName: "TerraformProviderVersionListPaginationQuery") {
        node(id: $providerId) {
            ...on TerraformProvider {
                versions(
                    after: $after
                    before: $before
                    first: $first
                    last: $last
                    sort: CREATED_AT_DESC
                ) @connection(key: "TerraformProviderVersionList_versions") {
                    totalCount
                    edges {
                        node {
                            id
                            ...TerraformProviderVersionListItemFragment_version
                        }
                    }
                }
            }
        }
      }
    `, queryData);

    return (
        <Box>
            <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                    <Typography variant="subtitle1">
                        {data.node?.versions?.edges?.length} version{data.node?.versions?.edges?.length === 1 ? '' : 's'}
                    </Typography>
                </Box>
            </Paper>
            <InfiniteScroll
                dataLength={data.node?.versions?.edges?.length ?? 0}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List disablePadding>
                    {data.node?.versions?.edges?.map((edge: any) => <TerraformProviderVersionListItem
                        key={edge.node.id}
                        fragmentRef={edge.node}
                    />)}
                </List>
            </InfiniteScroll>
        </Box>
    )
}

export default TerraformProviderVersionList
