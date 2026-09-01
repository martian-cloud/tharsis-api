import { List } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import InfiniteScroll from 'react-infinite-scroll-component';
import { LoadMoreFn, useFragment } from "react-relay/hooks";
import ListSkeleton from '../skeletons/ListSkeleton';
import ActivityEventTargetNotFound from './targets/ActivityEventTargetNotFound';
import ActivityEventGPGKeyTarget from './targets/ActivityEventGPGKeyTarget';
import ActivityEventGroupTarget from './targets/ActivityEventGroupTarget';
import ActivityEventManagedIdentityAccessRuleTarget from './targets/ActivityEventManagedIdentityAccessRule';
import ActivityEventManagedIdentityTarget from './targets/ActivityEventManagedIdentityTarget';
import ActivityEventCleanupPolicyTarget from './targets/ActivityEventCleanupPolicyTarget';
import ActivityEventNamespaceMembershipTarget from './targets/ActivityEventNamespaceMembershipTarget';
import ActivityEventPackageTarget from './targets/ActivityEventPackageTarget';
import ActivityEventPackageVersionTarget from './targets/ActivityEventPackageVersionTarget';
import ActivityEventPolicyTarget from './targets/ActivityEventPolicyTarget';
import ActivityEventRunTarget from './targets/ActivityEventRunTarget';
import ActivityEventRunGateTarget from './targets/ActivityEventRunGateTarget';
import ActivityEventServiceAccountTarget from './targets/ActivityEventServiceAccountTarget';
import ActivityEventStateVersionTarget from './targets/ActivityEventStateVersionTarget';
import ActivityEventTeamTarget from './targets/ActivityEventTeamTarget';
import ActivityEventRoleTarget from './targets/ActivityEventRoleTarget';
import ActivityEventTerraformModuleTarget from './targets/ActivityEventTerraformModuleTarget';
import ActivityEventTerraformModuleVersionTarget from './targets/ActivityEventTerraformModuleVersionTarget';
import ActivityEventTerraformProviderTarget from './targets/ActivityEventTerraformProviderTarget';
import ActivityEventTerraformProviderVersionTarget from './targets/ActivityEventTerraformProviderVersionTarget';
import ActivityEventVariableTarget from './targets/ActivityEventVariableTarget';
import ActivityEventVCSProviderTarget from './targets/ActivityEventVCSProviderTarget';
import ActivityEventWorkspaceTarget from './targets/ActivityEventWorkspaceTarget';
import ActivityEventRunnerTarget from './targets/ActivityEventRunnerTarget';
import ActivityEventFederatedRegistryTarget from './targets/ActivityEventFederatedRegistryTarget';
import ActivityEventTerraformProviderVersionMirrorTarget from './targets/ActivityEventTerraformProviderVersionMirrorTarget';
import { ActivityEventListFragment_connection$key } from './__generated__/ActivityEventListFragment_connection.graphql';

const TARGET_COMPONENT_MAP = {
    Workspace: ActivityEventWorkspaceTarget,
    Group: ActivityEventGroupTarget,
    ManagedIdentity: ActivityEventManagedIdentityTarget,
    NamespaceMembership: ActivityEventNamespaceMembershipTarget,
    CleanupPolicy: ActivityEventCleanupPolicyTarget,
    GPGKey: ActivityEventGPGKeyTarget,
    ManagedIdentityAccessRule: ActivityEventManagedIdentityAccessRuleTarget,
    ServiceAccount: ActivityEventServiceAccountTarget,
    NamespaceVariable: ActivityEventVariableTarget,
    Run: ActivityEventRunTarget,
    RunGate: ActivityEventRunGateTarget,
    StateVersion: ActivityEventStateVersionTarget,
    Team: ActivityEventTeamTarget,
    TerraformProvider: ActivityEventTerraformProviderTarget,
    TerraformProviderVersion: ActivityEventTerraformProviderVersionTarget,
    TerraformModule: ActivityEventTerraformModuleTarget,
    TerraformModuleVersion: ActivityEventTerraformModuleVersionTarget,
    VCSProvider: ActivityEventVCSProviderTarget,
    Role: ActivityEventRoleTarget,
    Runner: ActivityEventRunnerTarget,
    FederatedRegistry: ActivityEventFederatedRegistryTarget,
    TerraformProviderVersionMirror: ActivityEventTerraformProviderVersionMirrorTarget,
    Package: ActivityEventPackageTarget,
    PackageVersion: ActivityEventPackageVersionTarget,
    Policy: ActivityEventPolicyTarget
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
                    ...ActivityEventCleanupPolicyTargetFragment_event
                    ...ActivityEventGPGKeyTargetFragment_event
                    ...ActivityEventManagedIdentityAccessRuleTargetFragment_event
                    ...ActivityEventServiceAccountTargetFragment_event
                    ...ActivityEventVariableTargetFragment_event
                    ...ActivityEventRunTargetFragment_event
                    ...ActivityEventStateVersionTargetFragment_event
                    ...ActivityEventTeamTargetFragment_event
                    ...ActivityEventTerraformProviderTargetFragment_event
                    ...ActivityEventTerraformProviderVersionTargetFragment_event
                    ...ActivityEventTerraformModuleTargetFragment_event
                    ...ActivityEventTerraformModuleVersionTargetFragment_event
                    ...ActivityEventVCSProviderTargetFragment_event
                    ...ActivityEventRoleTargetFragment_event
                    ...ActivityEventRunnerTargetFragment_event
                    ...ActivityEventFederatedRegistryTargetFragment_event
                    ...ActivityEventTerraformProviderVersionMirrorTargetFragment_event
                    ...ActivityEventRunGateTargetFragment_event
                    ...ActivityEventPackageTargetFragment_event
                    ...ActivityEventPackageVersionTargetFragment_event
                    ...ActivityEventPolicyTargetFragment_event
                    ...ActivityEventTargetNotFoundFragment_event
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
                    // A null target means the API would not resolve it -- deleted, or outside what this
                    // viewer may see. The event itself still reads, so it is rendered either way.
                    if (!edge.node.target) {
                        return <ActivityEventTargetNotFound key={edge.node.id} fragmentRef={edge.node} />;
                    }
                    const Target = TARGET_COMPONENT_MAP[edge.node.target.__typename];
                    return Target ? <Target key={edge.node.id} fragmentRef={edge.node} /> : null;
                })}
            </List>
        </InfiniteScroll>
    );
}

export default ActivityEventList;
