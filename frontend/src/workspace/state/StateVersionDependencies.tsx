import Box from '@mui/material/Box';
import List from '@mui/material/List';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import SearchInput from '../../common/SearchInput';
import StateVersionDependencyListItem from './StateVersionDependencyListItem';
import { StateVersionDependenciesFragment_dependencies$key } from './__generated__/StateVersionDependenciesFragment_dependencies.graphql';

interface Props {
    fragmentRef: StateVersionDependenciesFragment_dependencies$key
}

const dependencySearchFilter = (search: string) => (dependency: any) => {
    return dependency.workspacePath.toLowerCase().includes(search);
};

function StateVersionDependencies(props: Props) {
    const { fragmentRef } = props;

    const data = useFragment<StateVersionDependenciesFragment_dependencies$key>(
        graphql`
        fragment StateVersionDependenciesFragment_dependencies on StateVersion
        {
            dependencies {
                workspacePath
                ...StateVersionDependencyListItemFragment_dependency
            }
        }
      `, fragmentRef)

    const [search, setSearch] = useState('');

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
    };

    const filteredDependencies = search ? data.dependencies.filter(dependencySearchFilter(search)) : data.dependencies;

    return (
        <Box>
            {data.dependencies.length > 0 && <SearchInput
                fullWidth
                placeholder="search for dependencies"
                onChange={onSearchChange}
            />}
            {(filteredDependencies.length === 0 && search !== '') && <Typography sx={{ padding: 2, marginTop: 4 }} align="center" color="textSecondary">
                No dependencies matching search <strong>{search}</strong>
            </Typography>}
            {(filteredDependencies.length === 0 && search === '') && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography color="textSecondary" align="center">
                        This workspace does not have any workspace dependencies
                    </Typography>
                </Box>
            </Paper>}
            {filteredDependencies.length > 0 && <Box marginTop={2}>
                <Paper>
                    <Box padding={'8px 16px'} display="flex" alignItems="center">
                        <Typography variant="subtitle1">{filteredDependencies.length} dependenc{filteredDependencies.length === 1 ? 'y' : 'ies'}</Typography>
                    </Box>
                </Paper>
                <List disablePadding>
                    {filteredDependencies.map((v: any) => <StateVersionDependencyListItem
                        key={v.workspacePath}
                        fragmentRef={v}
                    />)}
                </List>
            </Box>}
        </Box>
    );
}

export default StateVersionDependencies;