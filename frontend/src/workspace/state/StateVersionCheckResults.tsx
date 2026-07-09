import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useMemo, useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import SearchInput from '../../common/SearchInput';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import StateVersionCheckResultRow from './StateVersionCheckResultRow';
import { StateVersionCheckResultsFragment_checkResults$key } from './__generated__/StateVersionCheckResultsFragment_checkResults.graphql';

interface Props {
    fragmentRef: StateVersionCheckResultsFragment_checkResults$key
}

const searchFilter = (search: string) => (check: { readonly name: string; readonly status: string }) => {
    return check.name.toLowerCase().includes(search) || check.status.toLowerCase().includes(search);
};

const columns = [
    { label: 'Status' },
    { label: 'Name' },
    { label: 'Details' },
];

function StateVersionCheckResults(props: Props) {
    const { fragmentRef } = props;

    const data = useFragment<StateVersionCheckResultsFragment_checkResults$key>(
        graphql`
        fragment StateVersionCheckResultsFragment_checkResults on StateVersionInventory
        {
            checkResults {
                name
                status
                ...StateVersionCheckResultRowFragment_checkResult
            }
        }
      `, fragmentRef);

    const [search, setSearch] = useState('');

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
    };

    const filteredChecks = useMemo(() => {
        return search ? data.checkResults.filter(searchFilter(search)) : data.checkResults;
    }, [data.checkResults, search]);

    return (
        <Box>
            {data.checkResults.length > 0 && <SearchInput
                fullWidth
                placeholder="search for checks by name or status"
                onChange={onSearchChange}
            />}
            {(filteredChecks.length === 0 && search !== '') && <Typography sx={{ padding: 2, marginTop: 4 }} align="center" color="textSecondary">
                No checks matching search <strong>{search}</strong>
            </Typography>}
            {(filteredChecks.length === 0 && search === '') && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center' }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography color="textSecondary" align="center">
                        This workspace does not have any check results
                    </Typography>
                </Box>
            </Paper>}
            {filteredChecks.length > 0 && <ResponsiveTable columns={columns} ariaLabel="check results">
                {filteredChecks.map((check) => <StateVersionCheckResultRow
                    key={check.name}
                    fragmentRef={check}
                />)}
            </ResponsiveTable>}
        </Box>
    );
}

export default StateVersionCheckResults;
