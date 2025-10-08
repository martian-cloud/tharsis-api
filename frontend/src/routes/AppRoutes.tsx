import { Route, Routes } from 'react-router-dom';
import GraphiQLEditor from '../graphiql/GraphiQLEditor';
import NewGroup from '../groups/NewGroup';
import NewWorkspace from '../workspace/NewWorkspace';
import ExploreGroupsEntryPoint from './ExploreGroupsEntryPoint';
import GroupOrWorkspaceDetailsEntryPoint from './GroupOrWorkspaceDetailsEntryPoint';
import HomeEntryPoint from './HomeEntryPoint';
import ScrollRestoration from './ScrollRestoration';
import TerraformModuleSearchEntryPoint from './TerraformModuleSearchEntryPoint';
import TerraformModuleVersionDetailsEntryPoint from './TerraformModuleVersionDetailsEntryPoint';
import TerraformProviderSearchEntryPoint from './TerraformProviderSearchEntryPoint';
import TerraformProviderVersionDetailsEntryPoint from './TerraformProviderVersionDetailsEntryPoint';
import WorkspaceSearchEntryPoint from './WorkspaceSearchEntryPoint';
import AdminAreaEntryPoint from '../admin/AdminArea';
import UserPreferencesEntryPoint from './UserPreferencesEntryPoint';
import TeamDetailsEntryPoint from './TeamDetailsEntryPoint';

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
                <Route path="/preferences" element={<UserPreferencesEntryPoint /> } />
                <Route path="/teams/:teamName" element={<TeamDetailsEntryPoint />} />
                <Route path="/" element={<HomeEntryPoint />} />
            </Routes>
        </ScrollRestoration>
    );
}

export default AppRoutes;
