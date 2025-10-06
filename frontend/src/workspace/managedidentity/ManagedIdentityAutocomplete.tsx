import { Box, Typography } from '@mui/material';
import Autocomplete from '@mui/material/Autocomplete';
import CircularProgress from '@mui/material/CircularProgress';
import TextField from '@mui/material/TextField';
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useEffect, useState } from 'react';
import { useRelayEnvironment } from "react-relay/hooks";
import { fetchQuery } from 'relay-runtime';
import ManagedIdentityTypeChip from '../../groups/managedidentity/ManagedIdentityTypeChip';
import { ManagedIdentityAutocompleteQuery } from './__generated__/ManagedIdentityAutocompleteQuery.graphql';

export interface ManagedIdentityOption {
    readonly id: string;
    readonly name: string;
    readonly description: string;
    readonly resourcePath: string;
    readonly groupPath: string;
    readonly type: string;
}

interface Props {
    namespacePath: string
    value: ManagedIdentityOption | null
    assignedManagedIdentityIDs: any
    onSelected: (value: ManagedIdentityOption | null) => void
}

function ManagedIdentityAutocomplete(props: Props) {
    const { namespacePath, value, assignedManagedIdentityIDs, onSelected } = props;

    const [options, setOptions] = useState<ReadonlyArray<ManagedIdentityOption> | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState('');

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly ManagedIdentityOption[]) => void,
                ) => {
                    fetchQuery<ManagedIdentityAutocompleteQuery>(
                        environment,
                        graphql`
                          query ManagedIdentityAutocompleteQuery($path: String!, $search: String!) {
                            namespace(fullPath: $path) {
                                managedIdentities(first: 50, includeInherited: true, search: $search, sort: GROUP_LEVEL_DESC) {
                                    edges {
                                        node {
                                            id
                                            name
                                            groupPath
                                            resourcePath
                                            description
                                            type
                                        }
                                    }
                                }
                            }
                          }
                        `,
                        { path: namespacePath, search: request.input },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async response => {
                        const options = response?.namespace?.managedIdentities?.edges?.map(edge => edge?.node as ManagedIdentityOption);
                        callback(options);
                    });
                },
                300,
            ),
        [environment, namespacePath],
    );

    useEffect(() => {
        let active = true;

        setLoading(true);

        fetch({ input: inputValue }, (results?: readonly ManagedIdentityOption[]) => {
            if (active) {
                setOptions(results ?? []);
                setLoading(false);
            }
        });

        return () => {
            active = false;
        };
    }, [fetch, inputValue]);

    return (
        <Autocomplete
            fullWidth
            value={value}
            onChange={(event: React.SyntheticEvent, value: ManagedIdentityOption | null) => onSelected(value)}
            onInputChange={(_, newInputValue: string) => setInputValue(newInputValue)}
            filterOptions={(options) => options.filter(option => !assignedManagedIdentityIDs.has(option.id))}
            isOptionEqualToValue={(option: ManagedIdentityOption, value: ManagedIdentityOption) => option.id === value.id}
            getOptionLabel={(option: ManagedIdentityOption) => option.resourcePath}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: ManagedIdentityOption, { inputValue }) => {
                const matches = match(option.name, inputValue);
                const parts = parse(option.name, matches);
                return (
                    <Box component="li" {...props}>
                        <Box width="100%" display="flex" justifyContent="space-between" alignItems="center">
                            <Box>
                                <Typography>
                                    {parts.map((part: any, index: number) => (
                                        <span
                                            key={index}
                                            style={{
                                                fontWeight: part.highlight ? 700 : 400,
                                            }}
                                        >
                                            {part.text}
                                        </span>
                                    ))}
                                </Typography>
                                <Typography variant="caption" color="textSecondary">{option.groupPath}</Typography>
                            </Box>
                            <ManagedIdentityTypeChip type={option.type} />
                        </Box>
                    </Box>
                );
            }}
            options={options ?? []}
            loading={loading}
            renderInput={(params) => (
                <TextField
                    {...params}
                    placeholder='Select a managed identity'
                    InputProps={{
                        ...params.InputProps,
                        endAdornment: (
                            <React.Fragment>
                                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                {params.InputProps.endAdornment}
                            </React.Fragment>
                        ),
                    }}
                />
            )}
        />
    )
}

export default ManagedIdentityAutocomplete;
