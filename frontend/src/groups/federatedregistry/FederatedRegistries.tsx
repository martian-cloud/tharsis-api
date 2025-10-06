import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import EditFederatedRegistry from './EditFederatedRegistry';
import FederatedRegistryDetails from './FederatedRegistryDetails';
import FederatedRegistryList from './FederatedRegistryList';
import NewFederatedRegistry from './NewFederatedRegistry';
import { FederatedRegistriesFragment_group$key } from './__generated__/FederatedRegistriesFragment_group.graphql';

interface Props {
    fragmentRef: FederatedRegistriesFragment_group$key;
}

function FederatedRegistries({ fragmentRef }: Props) {
    const data = useFragment<FederatedRegistriesFragment_group$key>(
        graphql`
        fragment FederatedRegistriesFragment_group on Group
        {
            ...FederatedRegistryListFragment_group
            ...FederatedRegistryDetailsFragment_group
            ...NewFederatedRegistryFragment_group
            ...EditFederatedRegistryFragment_group
        }
    `, fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<FederatedRegistryList fragmentRef={data} />} />
                <Route path="new" element={<NewFederatedRegistry fragmentRef={data} />} />
                <Route path=":id/edit" element={<EditFederatedRegistry fragmentRef={data} />} />
                <Route path=":id/*" element={<FederatedRegistryDetails fragmentRef={data} />} />
            </Routes>
        </Box>
    );
}

export default FederatedRegistries;
