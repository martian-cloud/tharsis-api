import React, { useEffect, useState } from "react";
import { Autocomplete, Box, CircularProgress, SxProps, TextField, Theme, Typography } from "@mui/material";
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import { fetchQuery, useRelayEnvironment } from "react-relay/hooks";
import graphql from 'babel-plugin-relay/macro';
import throttle from "lodash.throttle";
import { GroupAutocompleteQuery } from "./__generated__/GroupAutocompleteQuery.graphql";

export interface GroupOption {
    readonly id: string
    readonly name: string
    readonly description: string
    readonly fullPath: string
}

interface Props {
    placeholder: string
    includeNoParentOption?: boolean
    sx?: SxProps<Theme>
    onSelected: (value: GroupOption | null) => void
    filterGroups: (options: GroupOption[]) => GroupOption[]
}

function GroupAutocomplete(props: Props) {
    const { placeholder, includeNoParentOption, sx, onSelected, filterGroups } = props
    const [options, setOptions] = useState<ReadonlyArray<GroupOption>>([]);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState<string>('');

    const noParentOption = includeNoParentOption ? { id: '', name: '', description: '', fullPath: '<< no parent >>' } : null;

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly GroupOption[]) => void,
                ) => {
                    fetchQuery<GroupAutocompleteQuery>(
                        environment,
                        graphql`
                            query GroupAutocompleteQuery($search: String!) {
                                groups(first: 50, search: $search, sort: FULL_PATH_ASC) {
                                    edges {
                                        node {
                                            id
                                            name
                                            description
                                            fullPath
                                        }
                                    }
                                }
                            }
                        `,
                        { search: request.input },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async response => {
                        const options = response?.groups?.edges?.map(edge => edge?.node as GroupOption);
                        callback(options);
                    });
                },
                300,
            ),
        [environment, inputValue]);

    useEffect(() => {
        let active = true;

        setLoading(true);

        fetch({ input: inputValue }, (results?: readonly GroupOption[]) => {
            if (active) {
                setOptions(results ?? [])
                setLoading(false)
            }
        });

        return () => {
            active = false;
        };
    }, [fetch, inputValue])

    return (
        <Autocomplete
            sx={ sx }
            fullWidth
            size="small"
            onChange={(event: React.SyntheticEvent, value: GroupOption | null) => onSelected(value)}
            onInputChange={(_, newInputValue: string) => setInputValue(newInputValue)}
            filterOptions={filterGroups}
            isOptionEqualToValue={(option: GroupOption, value: GroupOption) => option.id === value.id}
            getOptionLabel={(option: GroupOption) => option.fullPath}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: any, { inputValue }) => {
                const matches = match(option.fullPath, inputValue);
                const parts = parse(option.fullPath, matches);
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
            options={
                noParentOption
                    ? [noParentOption, ...options]
                    : options ?? []}
            loading={loading}
            renderInput={(params) => (
                <TextField
                    {...params}
                    placeholder={placeholder}
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
    );
}

export default GroupAutocomplete;
