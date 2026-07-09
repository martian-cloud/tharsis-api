import DeleteForeverOutlinedIcon from '@mui/icons-material/DeleteForeverOutlined';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import SearchInput from '../../common/SearchInput';
import StateVersionResourceListItem from './StateVersionResourceListItem';
import { StateVersionResourcesFragment_resources$key } from './__generated__/StateVersionResourcesFragment_resources.graphql';

interface Props {
    fragmentRef: StateVersionResourcesFragment_resources$key
    destroyed?: boolean
}

const searchFilter = (search: string) => (resource: { readonly name: string; readonly type: string }) => {
    return resource.name.toLowerCase().includes(search) || resource.type.includes(search);
};

function StateVersionResources(props: Props) {
    const { fragmentRef, destroyed } = props;

    const data = useFragment<StateVersionResourcesFragment_resources$key>(
        graphql`
        fragment StateVersionResourcesFragment_resources on StateVersionInventory
        {
            resources {
                name
                provider
                type
                ...StateVersionResourceListItemFragment_resource
            }
        }
      `, fragmentRef)

    const [search, setSearch] = useState('');

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
    };

    const filteredOutputs = search ? data.resources.filter(searchFilter(search)) : data.resources;

    return (
        <Box>
            {data.resources.length > 0 && <SearchInput
                fullWidth
                placeholder="search for resources by name or type"
                onChange={onSearchChange}
            />}
            {(filteredOutputs.length === 0 && search !== '') && <Typography sx={{ padding: 2, marginTop: 4 }} align="center" color="textSecondary">
                No resources matching search <strong>{search}</strong>
            </Typography>}
            {(filteredOutputs.length === 0 && search === '' && destroyed) && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <DeleteForeverOutlinedIcon sx={{ fontSize: 48, color: 'runStatus.destroy', mb: 1 }} />
                    <Typography align="center" gutterBottom>
                        Workspace destroyed
                    </Typography>
                    <Typography color="textSecondary" align="center">
                        All resources in this workspace have been destroyed, so the workspace is now empty. Create a new run to provision resources again.
                    </Typography>
                </Box>
            </Paper>}
            {(filteredOutputs.length === 0 && search === '' && !destroyed) && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography color="textSecondary" align="center">
                        This workspace does not have any resources
                    </Typography>
                </Box>
            </Paper>}
            {filteredOutputs.length > 0 && <Box sx={{ mt: 2 }}>
                <ResponsiveTable
                    ariaLabel="resources"
                    columns={[
                        { label: 'Name' },
                        { label: 'Type' },
                        { label: 'Provider' },
                        { label: 'Module' },
                    ]}
                >
                    {filteredOutputs.map((r) => <StateVersionResourceListItem
                        key={`${r.provider}::${r.type}::${r.name}`}
                        fragmentRef={r}
                    />)}
                </ResponsiveTable>
            </Box>}
        </Box>
    );
}

export default StateVersionResources;
