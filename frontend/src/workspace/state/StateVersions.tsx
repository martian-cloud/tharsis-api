import { Box, Typography } from '@mui/material'
import graphql from 'babel-plugin-relay/macro'
import React from "react";
import { useFragment } from 'react-relay';
import { Route, Routes } from 'react-router-dom'
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import StateVersionDetails from './StateVersionDetails';
import StateVersionList from './StateVersionList';
import { StateVersionsFragment_stateVersions$key } from './__generated__/StateVersionsFragment_stateVersions.graphql'

interface Props {
    fragmentRef: StateVersionsFragment_stateVersions$key
}

function StateVersions(props: Props) {
    const data = useFragment<StateVersionsFragment_stateVersions$key>(
        graphql`
        fragment StateVersionsFragment_stateVersions on Workspace
        {
            fullPath
            ...StateVersionListFragment_workspace
            ...StateVersionDetailsFragment_details
        }
        `, props.fragmentRef
    );

    return (
        <Box>
            <Routes>
                <Route index element={<Box>
                    <NamespaceBreadcrumbs
                        namespacePath={data.fullPath}
                        childRoutes={[
                            { title: "state versions", path: 'state_versions' }
                        ]}
                    />
                    <Typography variant="h5">State Versions</Typography>
                    <StateVersionList fragmentRef={data} />
                </Box>} />
                <Route path={`:id/*`} element={<StateVersionDetails fragmentRef={data} />} />
            </Routes>
        </Box>
    );
}

export default StateVersions;
