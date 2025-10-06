import Box from '@mui/material/Box';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import ManagedIdentityDetails from './ManagedIdentityDetails';
import ManagedIdentityList from './ManagedIdentityList';
import NewManagedIdentity from './NewManagedIdentity';
import graphql from 'babel-plugin-relay/macro';
import { ManagedIdentitiesFragment_group$key } from './__generated__/ManagedIdentitiesFragment_group.graphql';
import EditManagedIdentity from './EditManagedIdentity';

interface Props {
    fragmentRef: ManagedIdentitiesFragment_group$key
}

function ManagedIdentities(props: Props) {
    const group = useFragment<ManagedIdentitiesFragment_group$key>(
        graphql`
        fragment ManagedIdentitiesFragment_group on Group
        {
            ...ManagedIdentityListFragment_group
            ...NewManagedIdentityFragment_group
            ...EditManagedIdentityFragment_group
            ...ManagedIdentityDetailsFragment_group
        }
      `, props.fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<ManagedIdentityList fragmentRef={group} />} />
                <Route path={`new`} element={<NewManagedIdentity fragmentRef={group} />} />
                <Route path={`:id/edit`} element={<EditManagedIdentity fragmentRef={group} />} />
                <Route path={`:id`} element={<ManagedIdentityDetails fragmentRef={group} />} />
            </Routes>
        </Box>
    );
}

export default ManagedIdentities;
