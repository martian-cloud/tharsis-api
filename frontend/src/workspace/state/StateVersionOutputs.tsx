import Box from '@mui/material/Box';
import Button from '@mui/material/Button'
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import SearchInput from '../../common/SearchInput';
import StateVersionOutputListItem from './StateVersionOutputListItem';
import { StateVersionOutputsFragment_outputs$key } from './__generated__/StateVersionOutputsFragment_outputs.graphql';

interface Props {
    fragmentRef: StateVersionOutputsFragment_outputs$key
}

const outputSearchFilter = (search: string) => (output: any) => {
    return output.name.toLowerCase().includes(search);
};

function StateVersionOutputs(props: Props) {
    const { fragmentRef } = props;

    const data = useFragment<StateVersionOutputsFragment_outputs$key>(
        graphql`
        fragment StateVersionOutputsFragment_outputs on StateVersion
        {
            outputs {
                name
                ...StateVersionOutputListItemFragment_output
            }
        }
      `, fragmentRef)

    const [showValues, setShowValues] = useState(false);
    const [search, setSearch] = useState('');

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
    };

    const filteredOutputs = search ? data.outputs.filter(outputSearchFilter(search)) : data.outputs;

    return (
        <Box>
            {data.outputs.length > 0 && <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                <SearchInput
                    fullWidth
                    placeholder="search for outputs"
                    onChange={onSearchChange}
                />
                <Button
                    size="small"
                    color="info"
                    sx={{ flexShrink: 0, minWidth: 200 }}
                    onClick={() => setShowValues(!showValues)}
                >
                    {showValues ? 'Hide Sensitive Values' : 'Show Sensitive Values'}
                </Button>
            </Stack>}
            {(filteredOutputs.length === 0 && search !== '') && <Typography sx={{ padding: 2, marginTop: 4 }} align="center" color="textSecondary">
                No outputs matching search <strong>{search}</strong>
            </Typography>}
            {(filteredOutputs.length === 0 && search === '') && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography color="textSecondary" align="center">
                        This run does not have any Terraform outputs
                    </Typography>
                </Box>
            </Paper>}
            {filteredOutputs.length > 0 && <Box sx={{ mt: 2 }}>
                <ResponsiveTable
                    ariaLabel="outputs"
                    columns={[
                        { label: 'Name' },
                        { label: 'Value' },
                    ]}
                >
                    {filteredOutputs.map((o: any) => <StateVersionOutputListItem
                        key={o.name}
                        fragmentRef={o}
                        showValues={showValues}
                    />)}
                </ResponsiveTable>
            </Box>}
        </Box>
    );
}

export default StateVersionOutputs;
