import React, { useEffect, useState } from "react";
import { Autocomplete, Box, CircularProgress, TextField, Typography } from "@mui/material";
import { useRelayEnvironment } from "react-relay/hooks";
import { fetchQuery } from 'relay-runtime';
import VCSProviderTypeChip from "../../../groups/vcsprovider/VCSProviderTypeChip";
import throttle from 'lodash.throttle';
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import graphql from 'babel-plugin-relay/macro';
import { VCSProviderAutocompleteQuery } from "./__generated__/VCSProviderAutocompleteQuery.graphql";

export interface VCSProviderOption {
    readonly id: string
    readonly label: string
    readonly description: string
    readonly type: string
}

interface Props {
    path: string
    value: any
    onSelected: (value: any) => void
}

function VCSProviderAutocomplete(props: Props) {
    const { path, value, onSelected } = props

    const [options, setOptions] = useState<ReadonlyArray<VCSProviderOption> | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState<string>('');

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly VCSProviderOption[]) => void,
                ) => {
                    fetchQuery<VCSProviderAutocompleteQuery>(
                        environment,
                        graphql`
                            query VCSProviderAutocompleteQuery($path: String!, $search: String!) {
                                workspace(fullPath: $path) {
                                    vcsProviders(first: 100, includeInherited: true, search: $search) {
                                        edges {
                                            node {
                                                ... on VCSProvider {
                                                    id
                                                    name
                                                    description
                                                    type
                                                    autoCreateWebhooks
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        `,
                        { path: path, search: request.input },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async (response: any) => {
                        const options = response?.workspace?.vcsProviders?.edges?.map((edge: { node: any }) => {
                            return { id: edge.node.id, label: edge.node.name, description: edge.node.description, type: edge.node.type }
                        })
                        callback(options);
                    });
                },
                300,
            ),
        [environment, path],
    );

    useEffect(() => {
        let active = true;

        setLoading(true);

        fetch({ input: inputValue }, (results?: readonly VCSProviderOption[]) => {
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
            sx={{ mb: 1 }}
            size="small"
            value={value}
            onChange={(event: React.SyntheticEvent, value: any) => onSelected(value)}
            inputValue={inputValue}
            isOptionEqualToValue={(option: VCSProviderOption, value: VCSProviderOption) => option.id === value.id}
            onInputChange={(event: React.SyntheticEvent<Element, Event>, newValue: string) => newValue ? setInputValue(newValue) : setInputValue('')}
            options={options ?? []}
            loading={loading}
            renderInput={(params) =>
                <TextField
                    {...params}
                    placeholder="VCS Provider"
                    label="VCS Provider"
                    InputProps={{
                        ...params.InputProps,
                        endAdornment: (
                            <React.Fragment>
                                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                {params.InputProps.endAdornment}
                            </React.Fragment>
                        ),
                    }}/>}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: VCSProviderOption, { inputValue }) => {
                const matches = match(option.label, inputValue)
                const parts = parse(option.label, matches)
                return (
                    <Box component="li" {...props}>
                        <Box width="100%" display="flex" justifyContent="space-between" alignItems="center">
                            <Box>
                                <Typography>
                                    {parts.map((part: { text: string, highlight: boolean }, index: number) => (
                                    <span
                                        key={index}
                                    >
                                        {part.text}
                                    </span>
                                ))}
                                </Typography>
                                <Typography variant="caption">{option.description}</Typography>
                            </Box>
                            <VCSProviderTypeChip type={option.type} />
                        </Box>
                    </Box>
                )
            }}
        />
    )
}

export default VCSProviderAutocomplete
