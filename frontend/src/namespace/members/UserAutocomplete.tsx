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
import { UserAutocompleteQuery } from './__generated__/UserAutocompleteQuery.graphql';

export interface UserOption {
    readonly id: string;
    readonly username: string;
    readonly email: string;
}

interface Props {
    onSelected: (value: UserOption | null) => void
}

function UserAutocomplete(props: Props) {
    const { onSelected } = props;
    
    const [options, setOptions] = useState<ReadonlyArray<UserOption> | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState('');

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly UserOption[]) => void,
                ) => {
                    fetchQuery<UserAutocompleteQuery>(
                        environment,
                        graphql`
                          query UserAutocompleteQuery($search: String!) {
                            users(first: 50, search: $search) {
                              edges {
                                  node {
                                      id
                                      username
                                      email
                                  }
                              }
                            }
                          }
                        `,
                        { search: request.input },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async response => {
                        const options = response?.users?.edges?.map(edge => edge?.node as UserOption);
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

        fetch({ input: inputValue }, (results?: readonly UserOption[]) => {
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
            onChange={(event: React.SyntheticEvent, value: UserOption | null) => onSelected(value)}
            onInputChange={(_, newInputValue: string) => setInputValue(newInputValue)}
            filterOptions={(x) => x}
            isOptionEqualToValue={(option: UserOption, value: UserOption) => option.id === value.id}
            getOptionLabel={(option: UserOption) => option.username}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: UserOption, { inputValue }) => {
                const matches = match(option.username, inputValue);
                const parts = parse(option.username, matches);
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
                            <Typography variant="caption">{option.email}</Typography>
                        </Box>
                    </Box>
                );
            }}
            options={options ?? []}
            loading={loading}
            renderInput={(params) => (
                <TextField
                    {...params}
                    placeholder='Select a user'
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

export default UserAutocomplete;
