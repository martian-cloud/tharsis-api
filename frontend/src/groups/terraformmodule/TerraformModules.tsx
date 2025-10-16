import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import TerraformModuleList from './TerraformModuleList';
import { TerraformModulesFragment_group$key } from './__generated__/TerraformModulesFragment_group.graphql';

interface Props {
    fragmentRef: TerraformModulesFragment_group$key
}

function TerraformModules(props: Props) {
    const data = useFragment<TerraformModulesFragment_group$key>(
        graphql`
        fragment TerraformModulesFragment_group on Group
        {
            ...TerraformModuleListFragment_group
        }
      `, props.fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<TerraformModuleList fragmentRef={data} />} />
            </Routes>
        </Box>
    );
}

export default TerraformModules;
