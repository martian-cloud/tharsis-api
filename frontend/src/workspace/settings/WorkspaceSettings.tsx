import { Box, Divider, styled, Typography } from '@mui/material';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { WorkspaceSettingsFragment_workspace$key } from './__generated__/WorkspaceSettingsFragment_workspace.graphql';
import WorkspaceGeneralSettings from './WorkspaceGeneralSettings';
import WorkspaceRunSettings from './WorkspaceRunSettings';
import WorkspaceDriftDetectionSettings from './WorkspaceDriftDetectionSettings';
import WorkspaceAdvancedSettings from './WorkspaceAdvancedSettings';
import WorkspaceVCSProviderSettings from './vcsprovider/WorkspaceVCSProviderSettings';
import WorkspaceStateSettings from './WorkspaceStateSettings';
import WorkspaceRunnerSettings from './WorkspaceRunnerSettings';

export const StyledDivider = styled(
    Divider
)(() => ({
    margin: "24px 0"
}))

interface Props {
    fragmentRef: WorkspaceSettingsFragment_workspace$key
}

function WorkspaceSettings(props: Props) {

    const data = useFragment(
        graphql`
        fragment WorkspaceSettingsFragment_workspace on Workspace
        {
            name
            description
            fullPath
            ...WorkspaceGeneralSettingsFragment_workspace
            ...WorkspaceRunnerSettingsFragment_workspace
            ...WorkspaceRunSettingsFragment_workspace
            ...WorkspaceDriftDetectionSettingsFragment_workspace
            ...WorkspaceAdvancedSettingsFragment_workspace
            ...WorkspaceVCSProviderSettingsFragment_workspace
            ...WorkspaceStateSettingsFragment_workspace
        }
    `, props.fragmentRef
    )

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[{ title: "settings", path: 'settings' }]} />
            <Typography marginBottom={4} variant="h5" gutterBottom>Workspace Settings</Typography>
            <StyledDivider />
            <WorkspaceGeneralSettings fragmentRef={data} />
            <StyledDivider />
            <WorkspaceRunnerSettings fragmentRef={data} />
            <StyledDivider />
            <WorkspaceRunSettings fragmentRef={data} />
            <StyledDivider />
            <WorkspaceDriftDetectionSettings fragmentRef={data} />
            <StyledDivider />
            <WorkspaceStateSettings fragmentRef={data} />
            <StyledDivider />
            <WorkspaceVCSProviderSettings fragmentRef={data} />
            <StyledDivider />
            <WorkspaceAdvancedSettings fragmentRef={data} />
        </Box>
    );
}

export default WorkspaceSettings;
