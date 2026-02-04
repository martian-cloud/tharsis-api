import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import ProviderMirrorList from './ProviderMirrorList';
import ProviderMirrorVersionDetails from './ProviderMirrorVersionDetails';
import { ProviderMirrorsFragment_namespace$key } from './__generated__/ProviderMirrorsFragment_namespace.graphql';

interface Props {
    fragmentRef: ProviderMirrorsFragment_namespace$key
}

function ProviderMirrors(props: Props) {
    const data = useFragment<ProviderMirrorsFragment_namespace$key>(
        graphql`
        fragment ProviderMirrorsFragment_namespace on Namespace
        {
            fullPath
        }
      `, props.fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<ProviderMirrorList namespacePath={data.fullPath} />} />
                <Route path=":mirrorId" element={<ProviderMirrorVersionDetails namespacePath={data.fullPath} />} />
            </Routes>
        </Box>
    );
}

export default ProviderMirrors;
