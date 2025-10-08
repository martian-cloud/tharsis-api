import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import EditServiceAccount from './EditServiceAccount';
import NewServiceAccount from './NewServiceAccount';
import ServiceAccountDetails from './ServiceAccountDetails';
import ServiceAccountList from './ServiceAccountList';
import { ServiceAccountsFragment_group$key } from './__generated__/ServiceAccountsFragment_group.graphql';

interface Props {
    fragmentRef: ServiceAccountsFragment_group$key
}

function ServiceAccounts(props: Props) {
    const data = useFragment<ServiceAccountsFragment_group$key>(
        graphql`
        fragment ServiceAccountsFragment_group on Group
        {
            ...ServiceAccountListFragment_group
            ...ServiceAccountDetailsFragment_group
            ...NewServiceAccountFragment_group
            ...EditServiceAccountFragment_group
        }
      `, props.fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<ServiceAccountList fragmentRef={data} />} />
                <Route path={`new`} element={<NewServiceAccount fragmentRef={data} />} />
                <Route path={`:id`} element={<ServiceAccountDetails fragmentRef={data} />} />
                <Route path={`:id/edit`} element={<EditServiceAccount fragmentRef={data} />} />
            </Routes>
        </Box>
    );
}

export default ServiceAccounts;
