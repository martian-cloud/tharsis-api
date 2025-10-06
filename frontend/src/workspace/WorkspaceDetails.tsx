import { CircularProgress } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { Suspense, useMemo } from 'react';
import { useFragment, useSubscription } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import { ConnectionHandler, ConnectionInterface, GraphQLSubscriptionConfig, RecordSourceProxy } from 'relay-runtime';
import NamespaceActivity from '../namespace/activity/NamespaceActivity';
import NamespaceMemberships from '../namespace/members/NamespaceMemberships';
import Variables from '../namespace/variables/Variables';
import WorkspaceSettings from './settings/WorkspaceSettings';
import AssignedManagedIdentityList from './managedidentity/AssignedManagedIdentityList';
import { GetConnections } from './runs/RunList';
import Runs from './runs/Runs';
import StateVersions from './state/StateVersions';
import WorkspaceDetailsDrawer from './WorkspaceDetailsDrawer';
import WorkspaceDetailsIndex from './WorkspaceDetailsIndex';
import { WorkspaceDetailsFragment_workspace$key } from './__generated__/WorkspaceDetailsFragment_workspace.graphql';
import { WorkspaceDetailsRunSubscription, WorkspaceDetailsRunSubscription$data } from './__generated__/WorkspaceDetailsRunSubscription.graphql';
import { WorkspaceDetailsWorkspaceEventSubscription } from './__generated__/WorkspaceDetailsWorkspaceEventSubscription.graphql';

const runSubscription = graphql`subscription WorkspaceDetailsRunSubscription($input: RunSubscriptionInput!) {
  workspaceRunEvents(input: $input) {
    action
    run {
      id
      ...RunListItemFragment_run
      ...RunDetailsSidebarFragment_details
      ...RunDetailsPlanStageFragment_plan
      ...RunDetailsApplyStageFragment_apply
    }
  }
}`;

const workspaceSubscription = graphql`subscription WorkspaceDetailsWorkspaceEventSubscription($input: WorkspaceSubscriptionInput!) {
  workspaceEvents(input: $input) {
      action
      workspace {
          id
          ...WorkspaceDetailsIndexFragment_workspace
      }
  }
}`;

interface Props {
  fragmentRef: WorkspaceDetailsFragment_workspace$key
  route: string
}

function WorkspaceDetails(props: Props) {
  const { route, fragmentRef } = props;

  const data = useFragment<WorkspaceDetailsFragment_workspace$key>(
    graphql`
    fragment WorkspaceDetailsFragment_workspace on Workspace
    {
      id
      name
      description
      fullPath
      ...WorkspaceDetailsIndexFragment_workspace
      ...AssignedManagedIdentityListFragment_assignedManagedIdentities
      ...RunsFragment_runs
      ...StateVersionsFragment_stateVersions
      ...VariablesFragment_variables
      ...NamespaceMembershipsFragment_memberships
      ...WorkspaceSettingsFragment_workspace
      ...NamespaceActivityFragment_activity
    }
`, fragmentRef);

  const runSubscriptionConfig = useMemo<GraphQLSubscriptionConfig<WorkspaceDetailsRunSubscription>>(() => ({
    variables: { input: { workspacePath: data.fullPath } },
    subscription: runSubscription,
    onCompleted: () => console.log("Subscription completed"),
    onError: () => console.warn("Subscription error"),
    updater: (store: RecordSourceProxy, payload: WorkspaceDetailsRunSubscription$data | null | undefined) => {
      if (!payload) {
        return;
      }
      const record = store.get(payload.workspaceRunEvents.run.id);
      if (record == null) {
        return;
      }
      GetConnections(data.id).forEach(id => {
        const connectionRecord = store.get(id);
        if (connectionRecord) {
          const { NODE, EDGES } = ConnectionInterface.get();

          const recordId = record.getDataID();
          // Check if edge already exists in connection
          const nodeAlreadyExistsInConnection = connectionRecord
            .getLinkedRecords(EDGES)
            ?.some(
              edge => edge?.getLinkedRecord(NODE)?.getDataID() === recordId,
            );
          if (!nodeAlreadyExistsInConnection) {
            // Create Edge
            const edge = ConnectionHandler.createEdge(
              store,
              connectionRecord,
              record,
              'RunEdge'
            );
            if (edge) {
              // Add edge to the beginning of the connection
              ConnectionHandler.insertEdgeBefore(
                connectionRecord,
                edge,
              );
            }
          }
        }
      });
    }
  }), [data.fullPath, data.id]);

  const wsSubscriptionConfig = useMemo<GraphQLSubscriptionConfig<WorkspaceDetailsWorkspaceEventSubscription>>(() => ({
    variables: { input: { workspacePath: data.fullPath } },
    subscription: workspaceSubscription,
    onCompleted: () => console.log("Subscription completed"),
    onError: () => console.warn("Subscription error")
  }), [data.fullPath]);

  useSubscription<WorkspaceDetailsRunSubscription>(runSubscriptionConfig);
  useSubscription<WorkspaceDetailsWorkspaceEventSubscription>(wsSubscriptionConfig);

  const workspacePath = data.fullPath;

  return (
    <Box display="flex">
      <WorkspaceDetailsDrawer workspaceName={data.name} workspacePath={workspacePath} route={route} />
      <Box component="main" flexGrow={1}>
        <Suspense fallback={<Box
          sx={{
            width: '100%',
            height: `calc(100vh - 64px)`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}
        >
          <CircularProgress />
        </Box>}>
          <Box maxWidth={1200} margin="auto" padding={2}>
            <Routes>
              <Route path={`${workspacePath}/*`} element={<WorkspaceDetailsIndex fragmentRef={data} />} />
              <Route path={`${workspacePath}/-/activity/*`} element={<NamespaceActivity fragmentRef={data} />} />
              <Route path={`${workspacePath}/-/runs/*`} element={<Runs fragmentRef={data} />} />
              <Route path={`${workspacePath}/-/state_versions/*`} element={<StateVersions fragmentRef={data} />} />
              <Route path={`${workspacePath}/-/managed_identities/*`} element={<AssignedManagedIdentityList fragmentRef={data} />} />
              <Route path={`${workspacePath}/-/variables/*`} element={<Variables fragmentRef={data} />} />
              <Route path={`${workspacePath}/-/members/*`} element={<NamespaceMemberships fragmentRef={data} />} />
              <Route path={`${workspacePath}/-/settings/*`} element={<WorkspaceSettings fragmentRef={data} />} />
            </Routes>
          </Box>
        </Suspense>
      </Box>
    </Box>
  );
}

export default WorkspaceDetails;
