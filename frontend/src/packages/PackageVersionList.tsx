import NoResults from '@/common/NoResults';
import { Box, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { usePaginationFragment } from "react-relay/hooks";
import ListSkeleton from '../skeletons/ListSkeleton';
import PackageVersionListItem from './PackageVersionListItem';
import { PackageVersionListFragment_package$key } from './__generated__/PackageVersionListFragment_package.graphql';
import { PackageVersionListPaginationQuery } from './__generated__/PackageVersionListPaginationQuery.graphql';

interface Props {
    fragmentRef: PackageVersionListFragment_package$key
    onSelectVersion?: (versionId: string) => void
}

// The history picks which version to view. Editing and deleting a version are header actions, so this
// list is read-only and the same on the group and registry pages.
function PackageVersionList(props: Props) {
    const { onSelectVersion } = props;
    const theme = useTheme();

    const { data, loadNext, hasNext } = usePaginationFragment<PackageVersionListPaginationQuery, PackageVersionListFragment_package$key>(
        graphql`
        fragment PackageVersionListFragment_package on Package
        @refetchable(queryName: "PackageVersionListPaginationQuery") {
            id
            versions(
                after: $after
                before: $before
                first: $first
                last: $last
                sort: CREATED_AT_DESC
            ) @connection(key: "PackageVersionList_versions") {
                edges {
                    node {
                        id
                        ...PackageVersionListItemFragment_version
                    }
                }
            }
        }
        `, props.fragmentRef);

    const edges = data.versions?.edges ?? [];

    if (edges.length === 0) {
        return <NoResults sx={{ mt: 2 }}>This package does not have any versions.</NoResults>;
    }

    return (
        <Box>
            <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                    <Typography variant="subtitle1">
                        {edges.length} version{edges.length === 1 ? '' : 's'}
                    </Typography>
                </Box>
            </Paper>
            <InfiniteScroll
                dataLength={edges.length}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <List disablePadding>
                    {edges.map((edge) => edge?.node && <PackageVersionListItem
                        key={edge.node.id}
                        fragmentRef={edge.node}
                        onSelect={onSelectVersion}
                    />)}
                </List>
            </InfiniteScroll>
        </Box>
    );
}

export default PackageVersionList;
