import { Alert, AlertTitle, Box, Link, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash/throttle';
import { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { Link as RouterLink } from 'react-router-dom';
import { fetchQuery, useLazyLoadQuery, usePaginationFragment, useRelayEnvironment } from "react-relay/hooks";
import SearchInput from '../../common/SearchInput';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import ProviderMirrorListItem from './ProviderMirrorListItem';
import { ProviderMirrorListFragment_mirrors$key } from './__generated__/ProviderMirrorListFragment_mirrors.graphql';
import { ProviderMirrorListPaginationQuery } from './__generated__/ProviderMirrorListPaginationQuery.graphql';
import { ProviderMirrorListQuery } from './__generated__/ProviderMirrorListQuery.graphql';

const DESCRIPTION = 'Workspaces use a pull-through cache to add external providers to the mirror when mirroring is enabled.';
const INITIAL_ITEM_COUNT = 25;

const query = graphql`
    query ProviderMirrorListQuery($first: Int, $last: Int, $after: String, $before: String, $namespacePath: String!, $search: String) {
        ...ProviderMirrorListFragment_mirrors
    }
`;

interface Props {
    namespacePath: string
}

function ProviderMirrorList(props: Props) {
    const theme = useTheme();
    const [search, setSearch] = useState<string | undefined>();

    const { namespacePath } = props;

    const queryData = useLazyLoadQuery<ProviderMirrorListQuery>(query, { first: INITIAL_ITEM_COUNT, namespacePath }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext, refetch } = usePaginationFragment<ProviderMirrorListPaginationQuery, ProviderMirrorListFragment_mirrors$key>(
        graphql`
      fragment ProviderMirrorListFragment_mirrors on Query
      @refetchable(queryName: "ProviderMirrorListPaginationQuery") {
            namespace(fullPath: $namespacePath) {
                id
                __typename
                providerMirrorEnabled { value }
                terraformProviderMirrors(
                    after: $after
                    before: $before
                    first: $first
                    last: $last
                    sort: TYPE_ASC
                    search: $search
                ) @connection(key: "ProviderMirrorList_terraformProviderMirrors") {
                    totalCount
                    edges {
                        node {
                            id
                            version
                            groupPath
                            providerAddress
                            ...ProviderMirrorListItemFragment_mirror
                        }
                    }
                }
            }
      }
    `, queryData);

    const environment = useRelayEnvironment();

    const fetch = useMemo(
        () =>
            throttle(
                (input?: string) => {
                    fetchQuery(environment, query, { first: INITIAL_ITEM_COUNT, namespacePath: namespacePath, search: input })
                        .subscribe({
                            complete: () => {
                                setSearch(input);
                                refetch({ first: INITIAL_ITEM_COUNT, search: input }, { fetchPolicy: 'store-only' });
                            }
                        });
                },
                2000,
                { leading: false, trailing: true }
            ),
        [environment, refetch, namespacePath],
    );

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        fetch(event.target.value.toLowerCase());
    };

    const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const hasResults = data?.namespace?.terraformProviderMirrors?.edges?.length !== 0;
    const showList = hasResults || search;
    const mirrorDisabled = data?.namespace?.providerMirrorEnabled?.value === false;
    const settingsLink = `/groups/${namespacePath}/-/settings`;
    const isRootGroup = !namespacePath.includes('/');

    return (
        <Box>
            <NamespaceBreadcrumbs namespacePath={namespacePath} childRoutes={[{ title: "provider_mirror", path: 'provider_mirror' }]} />
            {showList && <Box>
                {mirrorDisabled && hasResults && <Alert severity="warning" variant="outlined" sx={{ mb: 2 }}><AlertTitle>Provider mirroring is disabled</AlertTitle>Runs will download providers directly from upstream registries. Update in <Link component={RouterLink} to={settingsLink}>Provider Mirror Settings</Link>.</Alert>}
                <Box marginBottom={2}>
                    <Typography variant="h5" gutterBottom>Terraform Provider Mirror</Typography>
                    <Typography variant="body2">{DESCRIPTION}{isRootGroup ? '' : ' Managed at root group.'}</Typography>
                </Box>
                <SearchInput
                    sx={{ marginBottom: 2 }}
                    placeholder="search for providers"
                    fullWidth
                    onChange={onSearchChange}
                    onKeyDown={onKeyDown}
                />
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {data?.namespace?.terraformProviderMirrors?.totalCount} provider{data?.namespace?.terraformProviderMirrors?.totalCount === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                {!hasResults && search && <Typography sx={{ mt: 2 }} color="textSecondary" align="center">No providers match your search</Typography>}
                <InfiniteScroll
                    dataLength={data?.namespace?.terraformProviderMirrors?.edges?.length ?? 0}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List disablePadding>
                        {data?.namespace?.terraformProviderMirrors?.edges?.map((edge: any) => (
                            <ProviderMirrorListItem
                                key={edge.node.id}
                                fragmentRef={edge.node}
                            />
                        ))}
                    </List>
                </InfiniteScroll>
            </Box>}
            {!showList && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6" gutterBottom>No provider mirrors available</Typography>
                    <Typography color="textSecondary" align="center">
                        {DESCRIPTION} Enable in <Link component={RouterLink} to={settingsLink}>Provider Mirror Settings</Link>.
                    </Typography>
                </Box>
            </Box>}
        </Box>
    );
}

export default ProviderMirrorList;
