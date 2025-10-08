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
import { ServiceAccountAutocompleteQuery } from './__generated__/ServiceAccountAutocompleteQuery.graphql';

export interface ServiceAccountOption {
    readonly id: string;
    readonly name: string;
    readonly resourcePath: string;
    readonly description: string
}

interface Props {
    namespacePath: string
    onSelected: (value: ServiceAccountOption | null) => void
}

function ServiceAccountAutocomplete(props: Props) {
    const {namespacePath, onSelected} = props;

    const [options, setOptions] = useState<ReadonlyArray<ServiceAccountOption> | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState('');

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly ServiceAccountOption[]) => void,
                ) => {
                    fetchQuery<ServiceAccountAutocompleteQuery>(
                        environment,
                        graphql`
                          query ServiceAccountAutocompleteQuery($path: String!, $search: String!) {
                            namespace(fullPath: $path) {
                                serviceAccounts(first: 50, includeInherited: true, search: $search) {
                                    edges {
                                        node {
                                            id
                                            name
                                            resourcePath
                                            description
                                        }
                                    }
                                }
                            }
                          }
                        `,
                        { path: namespacePath, search: request.input },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async response => {
                        const options = response?.namespace?.serviceAccounts?.edges?.map(edge => edge?.node as ServiceAccountOption);
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

        fetch({ input: inputValue }, (results?: readonly ServiceAccountOption[]) => {
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
            onChange={(event: React.SyntheticEvent, value: ServiceAccountOption | null) => onSelected(value)}
            onInputChange={(_, newInputValue: string) => setInputValue(newInputValue)}
            filterOptions={(x) => x}
            isOptionEqualToValue={(option: ServiceAccountOption, value: ServiceAccountOption) => option.id === value.id}
            getOptionLabel={(option: ServiceAccountOption) => option.resourcePath}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: ServiceAccountOption, { inputValue }) => {
                const matches = match(option.resourcePath, inputValue);
                const parts = parse(option.resourcePath, matches);
                return (
                    <Box component="li" {...props}>
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
                            <Typography variant="caption">{option.description}</Typography>
                        </Box>
                    </Box>
                );
            }}
            options={options ?? []}
            loading={loading}
            renderInput={(params) => (
                <TextField
                    {...params}
                    placeholder='Select a service account'
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

export default ServiceAccountAutocomplete;
