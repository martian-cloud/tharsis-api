import { Box, Button, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import NoResults from '../../common/NoResults';
import Link from '../../routes/Link';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import ListSkeleton from '../../skeletons/ListSkeleton';
import VCSEventListItem from './VCSEventListItem';
import { VCSEventListFragment_vcsEvents$key } from './__generated__/VCSEventListFragment_vcsEvents.graphql';
import { VCSEventListPaginationQuery } from './__generated__/VCSEventListPaginationQuery.graphql';
import { VCSEventListQuery } from './__generated__/VCSEventListQuery.graphql';
import { VCSProviderType } from './__generated__/VCSEventsFragment_vcsEvents.graphql';

const EMPTY_STATES = {
    linked: {
        title: 'No VCS events yet',
        description: "Events appear here once the repository sends a webhook, for example after a push, a tag, or a merge request update. If you've pushed and nothing shows up, check the VCS settings.",
        action: 'Check VCS Settings',
    },
    unlinked: {
        title: "This workspace isn't linked to a VCS provider",
        description: 'VCS events are recorded when a linked GitHub or GitLab repository sends Tharsis a push, tag, or merge request webhook. Link a repository to start seeing them here.',
        action: 'Go to VCS Settings',
    },
};

const query = graphql`
    query VCSEventListQuery($first: Int, $last: Int, $after: String, $before: String, $workspaceId: String!) {
        node(id: $workspaceId) {
            ...on Workspace @alias(as: "workspace") {
                ...VCSEventListFragment_vcsEvents
            }
        }
    }
`;

interface Props {
    workspaceId: string
    workspacePath: string
    // Null when the workspace isn't linked to a VCS provider.
    providerType: VCSProviderType | null | undefined
}

function VCSEventList({ workspaceId, workspacePath, providerType }: Props) {
    const queryData = useLazyLoadQuery<VCSEventListQuery>(query, { first: 50, workspaceId }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<VCSEventListPaginationQuery, VCSEventListFragment_vcsEvents$key>(
        graphql`
        fragment VCSEventListFragment_vcsEvents on Workspace
        @refetchable(queryName: "VCSEventListPaginationQuery") {
            vcsEvents(
                after: $after
                before: $before
                first: $first
                last: $last
                sort: CREATED_AT_DESC
            ) @connection(key: "VCSEventList_vcsEvents") {
                edges {
                    node {
                        id
                        ...VCSEventListItemFragment_vcsEvent
                    }
                }
            }
        }
        `, queryData.node?.workspace);

    const edges = data?.vcsEvents?.edges ?? [];

    if (edges.length === 0) {
        const emptyState = providerType ? EMPTY_STATES.linked : EMPTY_STATES.unlinked;
        return (
            <NoResults sx={{ mt: 2 }}>
                <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 1.5 }}>
                    <Typography color="textPrimary" sx={{ fontWeight: 500 }}>{emptyState.title}</Typography>
                    <Typography color="textSecondary" sx={{ maxWidth: 480 }}>{emptyState.description}</Typography>
                    <Button
                        size="small"
                        variant="outlined"
                        component={RouterLink}
                        to={`/groups/${workspacePath}/-/settings?section=vcs-provider`}
                    >
                        {emptyState.action}
                    </Button>
                </Box>
            </NoResults>
        );
    }

    return (
        <Box>
            <InfiniteScroll
                dataLength={edges.length}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <ResponsiveTable
                    ariaLabel="VCS events"
                    breakpoint="lg"
                    columns={[
                        { label: 'Status' },
                        { label: 'Type' },
                        { label: 'Reference' },
                        { label: 'Commit' },
                        { label: 'Created' },
                        { label: '' },
                    ]}
                >
                    {edges.map(edge => edge?.node && (
                        <VCSEventListItem key={edge.node.id} fragmentRef={edge.node} providerType={providerType} />
                    ))}
                </ResponsiveTable>
            </InfiniteScroll>
            <Typography variant="caption" color="textSecondary" component="p" sx={{ mt: 2 }}>
                Only webhook events that match this workspace's branch, tag regex, and merge request <Link color="inherit" underline="always" to={`/groups/${workspacePath}/-/settings?section=vcs-provider`}>settings</Link> are recorded.
                Runs started by these events appear on the <Link color="inherit" underline="always" to={`/groups/${workspacePath}/-/runs`}>Runs</Link> page.
            </Typography>
        </Box>
    );
}

export default VCSEventList;
