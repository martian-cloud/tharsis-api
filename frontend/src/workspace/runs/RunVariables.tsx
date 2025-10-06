import Box from '@mui/material/Box';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import React, { useMemo, useState } from 'react';
import RunVariableListItem from './RunVariableListItem';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { RunVariablesFragment_variables$key } from './__generated__/RunVariablesFragment_variables.graphql';
import TextField from '@mui/material/TextField';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { darken, useTheme } from '@mui/material/styles';
import Button from '@mui/material/Button';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { Dropdown } from '@mui/base/Dropdown';
import { IconButton, Menu, MenuItem } from '@mui/material';

interface Props {
    fragmentRef: RunVariablesFragment_variables$key
}

const variableSearchFilter = (search: string) => (variable: any) => {
    return variable.key.toLowerCase().startsWith(search);
};

function RunVariables(props: Props) {
    const { fragmentRef } = props;

    const theme = useTheme();

    const data = useFragment<RunVariablesFragment_variables$key>(
        graphql`
        fragment RunVariablesFragment_variables on Run
        {
            variables {
                key
                category
                namespacePath
                includedInTfConfig
                ...RunVariableListItemFragment_variable
            }
        }
      `, fragmentRef)

    const [showValues, setShowValues] = useState(false);
    const [showAllVariables, setShowAllVariables] = useState(false);
    const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
    const [variableCategory, setVariableCategory] = useState('terraform');
    const [search, setSearch] = useState('');

    const onVariableCategoryChange = (event: React.MouseEvent<HTMLElement>, newCategory: string) => {
        if (newCategory) {
            setVariableCategory(newCategory);
        }
    };

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
    };

    function onMenuOpen(event: React.MouseEvent<HTMLButtonElement>) {
        setMenuAnchorEl(event.currentTarget);
    }

    function onMenuClose() {
        setMenuAnchorEl(null);
    }

    const variables = useMemo(() => {
        return data.variables.filter((v: any) => {
            const categoryMatch = v.category === variableCategory;
            if (showAllVariables) {
                return categoryMatch;
            }
            return categoryMatch && (v.includedInTfConfig ?? true);
        });
    }, [data.variables, variableCategory, showAllVariables])

    const filteredVariables = useMemo(() => {
        return search ? variables.filter(variableSearchFilter(search)) : variables;
    }, [variables, search])

    return (
        <Box>
            <Box sx={{
                marginBottom: 2,
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                [theme.breakpoints.down('lg')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *:not(:last-child)': {
                        marginBottom: 2
                    },
                }
            }}>
                <ToggleButtonGroup
                    size="small"
                    color="primary"
                    value={variableCategory}
                    exclusive
                    onChange={onVariableCategoryChange}
                    sx={{ height: '100%' }}
                >
                    <ToggleButton value="terraform" size="small">Terraform</ToggleButton>
                    <ToggleButton value="environment" size="small">Environment</ToggleButton>
                </ToggleButtonGroup>
                <Box display="flex">
                    <Stack direction="row" spacing={2}>
                        <TextField
                            size="small"
                            margin='none'
                            placeholder="search for variables"
                            InputProps={{
                                sx: { background: darken(theme.palette.background.default, 0.5) }
                            }}
                            sx={{ width: 300, height: '100%' }}
                            onChange={onSearchChange}
                            autoComplete="off"
                        />
                        <Button
                            size="small"
                            color="info"
                            sx={{ height: '100%' }}
                            onClick={() => setShowValues(!showValues)}
                        >
                            {showValues ? 'Hide Values' : 'Show Values'}
                        </Button>
                    </Stack>
                    <Dropdown>
                        <IconButton
                            color="info"
                            size="small"
                            aria-label="more options menu"
                            aria-haspopup="menu"
                            onClick={onMenuOpen}
                        >
                            <MoreVertIcon />
                        </IconButton>
                        <Menu
                            id="variable-list-more-options-menu"
                            anchorEl={menuAnchorEl}
                            open={Boolean(menuAnchorEl)}
                            onClose={onMenuClose}
                        >
                            <MenuItem
                                onClick={() => {
                                    setShowAllVariables(!showAllVariables);
                                    onMenuClose();
                                }}>
                                {showAllVariables ? 'Hide Unused Variables' : 'Show All Variables'}
                            </MenuItem>
                        </Menu>
                    </Dropdown>
                </Box>
            </Box>
            {(filteredVariables.length === 0 && search !== '') && <Typography sx={{ padding: 2 }} align="center" color="textSecondary">
                No variables matching search
            </Typography>}
            {(filteredVariables.length === 0 && search === '') && <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center', marginBottom: 6 }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    {variableCategory === 'terraform' && <Typography color="textSecondary" align="center">
                        This run does not have any Terraform variables
                    </Typography>}
                    {variableCategory === 'environment' && <Typography color="textSecondary" align="center">
                        This run does not have any environment variables
                    </Typography>}
                </Box>
            </Paper>}
            {filteredVariables.length > 0 && <TableContainer>
                <Table sx={{ tableLayout: 'fixed' }}>
                    <TableHead>
                        <TableRow>
                            <TableCell>
                                <Typography color="textSecondary">Key</Typography>
                            </TableCell>
                            <TableCell>
                                <Typography color="textSecondary">Value</Typography>
                            </TableCell>
                            <TableCell>
                                <Typography color="textSecondary">Source</Typography>
                            </TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {filteredVariables.map((v: any) => <RunVariableListItem
                            key={v.key}
                            fragmentRef={v}
                            showValues={showValues}
                        />)}
                    </TableBody>
                </Table>
            </TableContainer>}
        </Box>
    );
}

export default RunVariables;
