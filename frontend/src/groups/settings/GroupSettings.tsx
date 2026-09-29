import { Box, CircularProgress, Divider, styled, Typography } from '@mui/material'
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs'
import graphql from 'babel-plugin-relay/macro'
import { Suspense } from 'react'
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import GroupGeneralSettings from './GroupGeneralSettings';
import GroupRunnerSettings from './GroupRunnerSettings';
import GroupAdvancedSettings from './GroupAdvancedSettings';
import { GroupSettingsFragment_group$key } from './__generated__/GroupSettingsFragment_group.graphql'
import { GroupSettingsQuery } from './__generated__/GroupSettingsQuery.graphql'
import { GroupSettingsContentFragment_group$key } from './__generated__/GroupSettingsContentFragment_group.graphql'
import GroupDriftDetectionSettings from './GroupDriftDetectionSettings';
import GroupProviderMirrorSettings from './GroupProviderMirrorSettings';
import GroupOutputVisibilitySettings from './GroupOutputVisibilitySettings';

interface Props {
    fragmentRef: GroupSettingsFragment_group$key
}

const StyledDivider = styled(
    Divider
)(() => ({
    margin: "24px 0"
}))

function GroupSettings(props: Props) {
    const group = useFragment<GroupSettingsFragment_group$key>(
        graphql`
        fragment GroupSettingsFragment_group on Group
        {
            id
            fullPath
        }
    `, props.fragmentRef
    )

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "settings", path: 'settings' },
                ]} />
            <Typography marginBottom={4} variant="h5" gutterBottom>Group Settings</Typography>
            <Suspense fallback={
                <Box padding={4} display="flex" justifyContent="center" alignItems="center">
                    <CircularProgress />
                </Box>
            }>
                <GroupSettingsContent groupId={group.id} />
            </Suspense>
        </Box>
    );
}

interface GroupSettingsContentProps {
    groupId: string
}

function GroupSettingsContent({ groupId }: GroupSettingsContentProps) {
    const queryData = useLazyLoadQuery<GroupSettingsQuery>(graphql`
        query GroupSettingsQuery($id: String!) {
            node(id: $id) {
                ...on Group {
                    ...GroupSettingsContentFragment_group
                }
            }
        }
    `, { id: groupId }, { fetchPolicy: 'store-and-network' });

    const data = useFragment<GroupSettingsContentFragment_group$key>(
        graphql`
        fragment GroupSettingsContentFragment_group on Group
        {
            ...GroupGeneralSettingsFragment_group
            ...GroupAdvancedSettingsFragment_group
            ...GroupRunnerSettingsFragment_group
            ...GroupDriftDetectionSettingsFragment_group
            ...GroupProviderMirrorSettingsFragment_group
            ...GroupOutputVisibilitySettingsFragment_group
        }
    `, queryData.node
    )

    if (!data) {
        return null;
    }

    return (
        <Box>
            <GroupGeneralSettings fragmentRef={data} />
            <StyledDivider />
            <GroupRunnerSettings fragmentRef={data} />
            <StyledDivider />
            <GroupDriftDetectionSettings fragmentRef={data} />
            <StyledDivider />
            <GroupProviderMirrorSettings fragmentRef={data} />
            <StyledDivider />
            <GroupOutputVisibilitySettings fragmentRef={data} />
            <StyledDivider />
            <GroupAdvancedSettings fragmentRef={data} />
        </Box>
    );
}

export default GroupSettings;
