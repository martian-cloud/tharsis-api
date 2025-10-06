import { Box, Divider, styled, Typography } from '@mui/material'
import React from 'react'
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs'
import graphql from 'babel-plugin-relay/macro'
import { useFragment } from 'react-relay/hooks';
import GroupGeneralSettings from './GroupGeneralSettings';
import GroupRunnerSettings from './GroupRunnerSettings';
import GroupAdvancedSettings from './GroupAdvancedSettings';
import { GroupSettingsFragment_group$key } from './__generated__/GroupSettingsFragment_group.graphql'
import GroupDriftDetectionSettings from './GroupDriftDetectionSettings';

interface Props {
    fragmentRef: GroupSettingsFragment_group$key
}

const StyledDivider = styled(
    Divider
)(() => ({
    margin: "24px 0"
}))

function GroupSettings(props: Props) {

    const data = useFragment(
        graphql`
        fragment GroupSettingsFragment_group on Group
        {
            fullPath
            ...GroupGeneralSettingsFragment_group
            ...GroupAdvancedSettingsFragment_group
            ...GroupRunnerSettingsFragment_group
            ...GroupDriftDetectionSettingsFragment_group
        }
    `, props.fragmentRef
    )

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[
                    { title: "settings", path: 'settings' },
                ]} />
            <Typography marginBottom={4} variant="h5" gutterBottom>Group Settings</Typography>
            <StyledDivider />
            <GroupGeneralSettings fragmentRef={data} />
            <StyledDivider />
            <GroupRunnerSettings fragmentRef={data}/>
            <StyledDivider />
            <GroupDriftDetectionSettings fragmentRef={data} />
            <StyledDivider />
            <GroupAdvancedSettings fragmentRef={data} />
        </Box>
    );
 }

   export default GroupSettings;
