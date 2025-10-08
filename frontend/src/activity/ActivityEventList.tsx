import { List } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { LoadMoreFn, useFragment } from "react-relay/hooks";
import ListSkeleton from '../skeletons/ListSkeleton';
import ActivityEventGPGKeyTarget from './targets/ActivityEventGPGKeyTarget';
import ActivityEventGroupTarget from './targets/ActivityEventGroupTarget';
import ActivityEventManagedIdentityAccessRuleTarget from './targets/ActivityEventManagedIdentityAccessRule';
import ActivityEventManagedIdentityTarget from './targets/ActivityEventManagedIdentityTarget';
import ActivityEventNamespaceMembershipTarget from './targets/ActivityEventNamespaceMembershipTarget';
import ActivityEventRunTarget from './targets/ActivityEventRunTarget';
import ActivityEventServiceAccountTarget from './targets/ActivityEventServiceAccountTarget';
import ActivityEventStateVersionTarget from './targets/ActivityEventStateVersionTarget';
import ActivityEventTeamTarget from './targets/ActivityEventTeamTarget';
import ActivityEventRoleTarget from './targets/ActivityEventRoleTarget';
import ActivityEventTerraformModuleTarget from './targets/ActivityEventTerraformModuleTarget';
import ActivityEventTerraformModuleVersionTarget from './targets/ActivityEventTerraformModuleVersionTarget';
import ActivityEventTerraformProviderTarget from './targets/ActivityEventTerraformProviderTarget';
import ActivityEventVariableTarget from './targets/ActivityEventVariableTarget';
import ActivityEventVCSProviderTarget from './targets/ActivityEventVCSProviderTarget';
import ActivityEventWorkspaceTarget from './targets/ActivityEventWorkspaceTarget';
import ActivityEventRunnerTarget from './targets/ActivityEventRunnerTarget';
import ActivityEventFederatedRegistryTarget from './targets/ActivityEventFederatedRegistryTarget';
import { ActivityEventListFragment_connection$key } from './__generated__/ActivityEventListFragment_connection.graphql';

const TARGET_COMPONENT_MAP = {
    Workspace: ActivityEventWorkspaceTarget,
    Group: ActivityEventGroupTarget,
    ManagedIdentity: ActivityEventManagedIdentityTarget,
    NamespaceMembership: ActivityEventNamespaceMembershipTarget,
    GPGKey: ActivityEventGPGKeyTarget,
    ManagedIdentityAccessRule: ActivityEventManagedIdentityAccessRuleTarget,
    ServiceAccount: ActivityEventServiceAccountTarget,
    NamespaceVariable: ActivityEventVariableTarget,
    Run: ActivityEventRunTarget,
    StateVersion: ActivityEventStateVersionTarget,
    Team: ActivityEventTeamTarget,
    TerraformProvider: ActivityEventTerraformProviderTarget,
    TerraformModule: ActivityEventTerraformModuleTarget,
    TerraformModuleVersion: ActivityEventTerraformModuleVersionTarget,
    VCSProvider: ActivityEventVCSProviderTarget,
    Role: ActivityEventRoleTarget,
    Runner: ActivityEventRunnerTarget,
    FederatedRegistry: ActivityEventFederatedRegistryTarget
} as any;

interface Props {
    fragmentRef: ActivityEventListFragment_connection$key
    loadNext: LoadMoreFn<any>
    hasNext: boolean
}

function ActivityEventList({ fragmentRef, loadNext, hasNext }: Props) {
    const data = useFragment<ActivityEventListFragment_connection$key>(graphql`
        fragment ActivityEventListFragment_connection on ActivityEventConnection {
            edges {
                node {
                    id
                    target {
                        __typename
                    }
                    ...ActivityEventWorkspaceTargetFragment_event
                    ...ActivityEventGroupTargetFragment_event
                    ...ActivityEventManagedIdentityTargetFragment_event
                    ...ActivityEventNamespaceMembershipTargetFragment_event
                    ...ActivityEventGPGKeyTargetFragment_event
                    ...ActivityEventManagedIdentityAccessRuleTargetFragment_event
                    ...ActivityEventServiceAccountTargetFragment_event
                    ...ActivityEventVariableTargetFragment_event
                    ...ActivityEventRunTargetFragment_event
                    ...ActivityEventStateVersionTargetFragment_event
                    ...ActivityEventTeamTargetFragment_event
                    ...ActivityEventTerraformProviderTargetFragment_event
                    ...ActivityEventTerraformModuleTargetFragment_event
                    ...ActivityEventTerraformModuleVersionTargetFragment_event
                    ...ActivityEventVCSProviderTargetFragment_event
                    ...ActivityEventRoleTargetFragment_event
                    ...ActivityEventRunnerTargetFragment_event
                    ...ActivityEventFederatedRegistryTargetFragment_event
                }
            }
        }
    `, fragmentRef);

    return (
        <InfiniteScroll
            dataLength={data.edges?.length ?? 0}
            next={() => loadNext(20)}
            hasMore={hasNext}
            loader={<ListSkeleton rowCount={3} />}
        >
            <List sx={{ paddingTop: 0 }}>
                {data.edges?.map((edge: any) => {
                    const Target = TARGET_COMPONENT_MAP[edge.node.target.__typename];
                    return Target ? <Target key={edge.node.id} fragmentRef={edge.node} /> : null;
                })}
            </List>
        </InfiniteScroll>
    );
}

export default ActivityEventList;
