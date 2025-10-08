import Box from '@mui/material/Box'
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay'
import { Route, Routes } from 'react-router-dom'
import EditVCSProvider from './EditVCSProvider';
import NewVCSProvider from './NewVCSProvider'
import VCSProviderDetails from './VCSProviderDetails'
import VCSProviderList from './VCSProviderList'
import EditVCSProviderOAuth from './EditVCSProviderOAuthCredentials';
import { VCSProvidersFragment_group$key } from './__generated__/VCSProvidersFragment_group.graphql'

interface Props {
    fragmentRef: VCSProvidersFragment_group$key
}

function VCSProviders(props: Props) {

    const data = useFragment<VCSProvidersFragment_group$key>(
        graphql`
        fragment VCSProvidersFragment_group on Group
        {
            ...VCSProviderListFragment_group
            ...NewVCSProviderFragment_group
            ...EditVCSProviderFragment_group
            ...VCSProviderDetailsFragment_group
            ...EditVCSProviderOAuthCredentialsFragment_group
        }
    `, props.fragmentRef)

    return (
        <Box>
            <Routes>
                <Route index element={<VCSProviderList fragmentRef={data} />} />
                <Route path={`new`} element={<NewVCSProvider fragmentRef={data} />} />
                <Route path={`:id`} element={<VCSProviderDetails fragmentRef={data} />} />
                <Route path={`:id/edit`} element={<EditVCSProvider fragmentRef={data} />} />
                <Route path={`:id/edit_oauth_credentials`} element={<EditVCSProviderOAuth fragmentRef={data} />} />
            </Routes>

        </Box>
    )
}

export default VCSProviders
