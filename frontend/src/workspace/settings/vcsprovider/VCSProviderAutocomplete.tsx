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
    readonly autoCreateWebhooks: boolean
}

interface Props {
    path: string
    value: VCSProviderOption | null
    onSelected: (value: VCSProviderOption | null) => void
    label?: string
    helperText?: string
    error?: boolean
    // excludeId hides a provider from the options, e.g. the one a link already uses.
    excludeId?: string
}

function VCSProviderAutocomplete(props: Props) {
    const { path, value, onSelected, label = 'VCS provider', helperText, error, excludeId } = props

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
                    ).toPromise().then(response => {
                        const options: VCSProviderOption[] = [];
                        response?.workspace?.vcsProviders?.edges?.forEach(edge => {
                            const node = edge?.node;
                            if (node && node.id !== excludeId) {
                                options.push({ id: node.id, label: node.name, description: node.description, type: node.type, autoCreateWebhooks: node.autoCreateWebhooks });
                            }
                        });
                        callback(options);
                    });
                },
                300,
            ),
        [environment, path, excludeId],
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
            size="small"
            value={value}
            onChange={(event: React.SyntheticEvent, value: VCSProviderOption | null) => onSelected(value)}
            inputValue={inputValue}
            isOptionEqualToValue={(option: VCSProviderOption, value: VCSProviderOption) => option.id === value.id}
            onInputChange={(event: React.SyntheticEvent<Element, Event>, newValue: string) => newValue ? setInputValue(newValue) : setInputValue('')}
            options={options ?? []}
            loading={loading}
            renderInput={(params) =>
                <TextField
                    {...params}
                    label={label}
                    helperText={helperText}
                    error={error}
                    slotProps={{
                        ...params.slotProps,

                        input: {
                            ...params.slotProps.input,
                            endAdornment: (
                                <React.Fragment>
                                    {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                    {params.slotProps.input.endAdornment}
                                </React.Fragment>
                            ),
                        }
                    }} />}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: VCSProviderOption, { inputValue }) => {
                const matches = match(option.label, inputValue)
                const parts = parse(option.label, matches)
                const { key, ...otherProps } = props as any;
                return (
                    <Box component="li" key={key} {...otherProps}>
                        <Box
                            sx={{
                                width: "100%",
                                display: "flex",
                                justifyContent: "space-between",
                                alignItems: "center"
                            }}>
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
                );
            }}
        />
    );
}

export default VCSProviderAutocomplete;
