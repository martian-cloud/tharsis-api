import { Box, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useFragment, useLazyLoadQuery, usePaginationFragment } from "react-relay/hooks";
import ListSkeleton from '../skeletons/ListSkeleton';
import TerraformModuleVersionListItem from './TerraformModuleVersionListItem';
import { TerraformModuleVersionListFragment_module$key } from './__generated__/TerraformModuleVersionListFragment_module.graphql';
import { TerraformModuleVersionListFragment_versions$key } from './__generated__/TerraformModuleVersionListFragment_versions.graphql';
import { TerraformModuleVersionListPaginationQuery } from './__generated__/TerraformModuleVersionListPaginationQuery.graphql';
import { TerraformModuleVersionListQuery } from './__generated__/TerraformModuleVersionListQuery.graphql';

const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query TerraformModuleVersionListQuery($first: Int, $last: Int, $after: String, $before: String, $moduleId: String!) {
        ...TerraformModuleVersionListFragment_versions
    }
`;

interface Props {
    fragmentRef: TerraformModuleVersionListFragment_module$key
}

function TerraformModuleVersionList(props: Props) {
    const theme = useTheme();
    const module = useFragment<TerraformModuleVersionListFragment_module$key>(
        graphql`
        fragment TerraformModuleVersionListFragment_module on TerraformModule
        {
          id
        }
    `, props.fragmentRef)

    const queryData = useLazyLoadQuery<TerraformModuleVersionListQuery>(query, { first: INITIAL_ITEM_COUNT, moduleId: module.id }, { fetchPolicy: 'store-and-network' })

    const { data, loadNext, hasNext } = usePaginationFragment<TerraformModuleVersionListPaginationQuery, TerraformModuleVersionListFragment_versions$key>(
        graphql`
      fragment TerraformModuleVersionListFragment_versions on Query
      @refetchable(queryName: "TerraformModuleVersionListPaginationQuery") {
        node(id: $moduleId) {
            ...on TerraformModule {
                versions(
                    after: $after
                    before: $before
                    first: $first
                    last: $last
                    sort: CREATED_AT_DESC
                ) @connection(key: "TerraformModuleVersionList_versions") {
                    totalCount
                    edges {
                        node {
                            id
                            ...TerraformModuleVersionListItemFragment_version
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
                    {data.node?.versions?.edges?.map((edge: any) => <TerraformModuleVersionListItem
                        key={edge.node.id}
                        fragmentRef={edge.node}
                    />)}
                </List>
            </InfiniteScroll>
        </Box>
    )
}

export default TerraformModuleVersionList
