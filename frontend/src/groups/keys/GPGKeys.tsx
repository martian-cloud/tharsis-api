import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import GPGKeyList from './GPGKeyList';
import NewGPGKey from './NewGPGKey';
import { GPGKeysFragment_group$key } from './__generated__/GPGKeysFragment_group.graphql';

interface Props {
    fragmentRef: GPGKeysFragment_group$key
}

function GPGKeys(props: Props) {
    const data = useFragment<GPGKeysFragment_group$key>(
        graphql`
        fragment GPGKeysFragment_group on Group
        {
            ...GPGKeyListFragment_group
            ...NewGPGKeyFragment_group
        }
      `, props.fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<GPGKeyList fragmentRef={data} />} />
                <Route path="new" element={<NewGPGKey fragmentRef={data} />} />
            </Routes>
        </Box>
    );
}

export default GPGKeys;
