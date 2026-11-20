import NoResults from '@/common/NoResults';
import DeleteForeverOutlinedIcon from '@mui/icons-material/DeleteForeverOutlined';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useMemo, useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { useFragment } from 'react-relay/hooks';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import SearchInput from '../../common/SearchInput';
import ListSkeleton from '../../skeletons/ListSkeleton';
import StateVersionResourceListItem from './StateVersionResourceListItem';
import { StateVersionResourcesFragment_resources$key } from './__generated__/StateVersionResourcesFragment_resources.graphql';

// A state can hold thousands of resources, so rows are rendered in pages as the user scrolls.
const PAGE_SIZE = 100;

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
                mode
                module
                name
                provider
                type
                ...StateVersionResourceListItemFragment_resource
            }
        }
      `, fragmentRef)

    const [search, setSearch] = useState('');
    const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
        setVisibleCount(PAGE_SIZE);
    };

    const filteredOutputs = useMemo(
        () => search ? data.resources.filter(searchFilter(search)) : data.resources,
        [data.resources, search]
    );

    const visibleResources = useMemo(() => filteredOutputs.slice(0, visibleCount), [filteredOutputs, visibleCount]);

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
                <Box
                    sx={{
                        padding: 4,
                        display: "flex",
                        flexDirection: "column",
                        justifyContent: "center",
                        alignItems: "center"
                    }}>
                    <DeleteForeverOutlinedIcon sx={{ fontSize: 48, color: 'runStatus.destroy', mb: 1 }} />
                    <Typography align="center" gutterBottom>
                        Workspace destroyed
                    </Typography>
                    <Typography color="textSecondary" align="center">
                        All resources in this workspace have been destroyed, so the workspace is now empty. Create a new run to provision resources again.
                    </Typography>
                </Box>
            </Paper>}
            {(filteredOutputs.length === 0 && search === '' && !destroyed) && <NoResults sx={{ mt: 4 }}>
                This workspace does not have any resources
            </NoResults>}
            {filteredOutputs.length > 0 && <Box sx={{ mt: 2 }}>
                <InfiniteScroll
                    dataLength={visibleResources.length}
                    next={() => setVisibleCount(count => count + PAGE_SIZE)}
                    hasMore={visibleResources.length < filteredOutputs.length}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <ResponsiveTable
                        ariaLabel="resources"
                        columns={[
                            { label: 'Name' },
                            { label: 'Type' },
                            { label: 'Provider' },
                            { label: 'Module' },
                        ]}
                    >
                        {visibleResources.map((r) => <StateVersionResourceListItem
                            key={`${r.module}::${r.mode}::${r.provider}::${r.type}::${r.name}`}
                            fragmentRef={r}
                        />)}
                    </ResponsiveTable>
                </InfiniteScroll>
            </Box>}
        </Box>
    );
}

export default StateVersionResources;
