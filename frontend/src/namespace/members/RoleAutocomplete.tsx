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
import { RoleAutocompleteQuery } from './__generated__/RoleAutocompleteQuery.graphql';

export interface RoleOption {
    readonly id: string;
    readonly name: string;
}

interface Props {
    onSelected: (value: RoleOption | null) => void
    size?: 'small' | 'medium'
}

function RoleAutocomplete(props: Props) {
    const { onSelected, size } = props;

    const [options, setOptions] = useState<ReadonlyArray<RoleOption> | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState('');

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly RoleOption[]) => void,
                ) => {
                    fetchQuery<RoleAutocompleteQuery>(
                        environment,
                        graphql`
                          query RoleAutocompleteQuery($search: String!) {
                            roles(first: 50, search: $search) {
                              edges {
                                  node {
                                      id
                                      name
                                  }
                              }
                            }
                          }
                        `,
                        { search: request.input },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async response => {
                        const options = response?.roles?.edges?.map(edge => edge?.node as RoleOption);
                        callback(options);
                    });
                },
                300,
            ),
        [environment],
    );

    useEffect(() => {
        let active = true;

        setLoading(true);

        fetch({ input: inputValue }, (results?: readonly RoleOption[]) => {
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
            size={size}
            fullWidth
            sx={{ minWidth: 150 }}
            onChange={(event: React.SyntheticEvent, value: RoleOption | null) => onSelected(value)}
            onInputChange={(_, newInputValue: string) => setInputValue(newInputValue)}
            filterOptions={(x) => x}
            isOptionEqualToValue={(option: RoleOption, value: RoleOption) => option.id === value.id}
            getOptionLabel={(option: RoleOption) => option.name}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: RoleOption, { inputValue }) => {
                const matches = match(option.name, inputValue);
                const parts = parse(option.name, matches);
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
                        </Box>
                    </Box>
                );
            }}
            options={options ?? []}
            loading={loading}
            renderInput={(params) => (
                <TextField
                    {...params}
                    placeholder='Select a role'
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

export default RoleAutocomplete;
