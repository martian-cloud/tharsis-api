import { Route, Routes } from 'react-router-dom';
import AdminAreaEntryPoint from '../admin/AdminArea';
import GraphiQLEditor from '../graphiql/GraphiQLEditor';
import NewGroup from '../groups/NewGroup';
import HomePage from '../home/HomePage';
import NewWorkspace from '../workspace/NewWorkspace';
import ExploreGroupsEntryPoint from './ExploreGroupsEntryPoint';
import GroupOrWorkspaceDetailsEntryPoint from './GroupOrWorkspaceDetailsEntryPoint';
import ScrollRestoration from './ScrollRestoration';
import TeamDetailsEntryPoint from './TeamDetailsEntryPoint';
import TerraformModuleSearchEntryPoint from './TerraformModuleSearchEntryPoint';
import TerraformModuleVersionDetailsEntryPoint from './TerraformModuleVersionDetailsEntryPoint';
import TerraformProviderSearchEntryPoint from './TerraformProviderSearchEntryPoint';
import TerraformProviderVersionDetailsEntryPoint from './TerraformProviderVersionDetailsEntryPoint';
import UserPreferencesEntryPoint from './UserPreferencesEntryPoint';
import WorkspaceSearchEntryPoint from './WorkspaceSearchEntryPoint';
import Teams from '../teams/Teams';

function AppRoutes() {
    return (
        <ScrollRestoration>
            <Routes>
                <Route path="/groups/-/new" element={<NewGroup />} />
                <Route path="/groups" element={<ExploreGroupsEntryPoint />} />
                <Route path="/workspaces" element={<WorkspaceSearchEntryPoint />} />
                <Route path="/workspaces/-/new" element={<NewWorkspace />} />
                <Route path="/graphiql" element={<GraphiQLEditor />} />
                <Route path="/groups/*" element={<GroupOrWorkspaceDetailsEntryPoint />} />
                <Route path="/provider-registry/:registryNamespace/:providerName/:version" element={<TerraformProviderVersionDetailsEntryPoint />} />
                <Route path="/provider-registry/:registryNamespace/:providerName" element={<TerraformProviderVersionDetailsEntryPoint />} />
                <Route path="/provider-registry/*" element={<TerraformProviderSearchEntryPoint />} />
                <Route path="/module-registry/:registryNamespace/:moduleName/:system/:version" element={<TerraformModuleVersionDetailsEntryPoint />} />
                <Route path="/module-registry/:registryNamespace/:moduleName/:system" element={<TerraformModuleVersionDetailsEntryPoint />} />
                <Route path="/module-registry/*" element={<TerraformModuleSearchEntryPoint />} />
                <Route path="/admin/*" element={<AdminAreaEntryPoint />} />
                <Route path="/preferences" element={<UserPreferencesEntryPoint />} />
                <Route path="/teams" element={<Teams />} />
                <Route path="/teams/:teamName" element={<TeamDetailsEntryPoint />} />
                <Route path="/" element={<HomePage />} />
            </Routes>
        </ScrollRestoration>
    );
}

export default AppRoutes;
