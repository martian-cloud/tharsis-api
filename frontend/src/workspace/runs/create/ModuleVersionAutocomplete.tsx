import { Autocomplete, Box, CircularProgress, TextField, Typography } from "@mui/material";
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useEffect, useState } from "react";
import { useRelayEnvironment } from "react-relay/hooks";
import { fetchQuery } from "relay-runtime";
import { ModuleVersionAutocompleteQuery } from "./__generated__/ModuleVersionAutocompleteQuery.graphql";

interface Props {
    registryNamespace: string
    moduleName: string
    system: string
    value: string | null
    onSelected: (value: string | null) => void
}

function ModuleVersionAutocomplete({ registryNamespace, moduleName, system, value, onSelected }: Props) {
    const environment = useRelayEnvironment();

    const [options, setOptions] = useState<string[] | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState('');

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: any) => void,
                ) => {
                    fetchQuery<ModuleVersionAutocompleteQuery>(
                        environment,
                        graphql`
                            query ModuleVersionAutocompleteQuery($registryNamespace: String!, $moduleName: String!, $system: String!, $versionSearch: String) {
                                terraformModule(registryNamespace: $registryNamespace, moduleName: $moduleName, system: $system) {
                                    versions(first: 50, search: $versionSearch, sort: CREATED_AT_DESC) {
                                        edges {
                                            node {
                                                version
                                            }
                                        }
                                    }
                                }
                            }
                        `, { registryNamespace, moduleName, system, versionSearch: request.input },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async response => {
                        const options = response?.terraformModule?.versions.edges?.map(edge => edge?.node?.version as string);
                        callback(options)
                    });
                },
                300
            ),
        [environment, registryNamespace, moduleName, system]
    );

    useEffect(() => {
        let active = true;

        setLoading(true);

        fetch({ input: inputValue }, (results?: string[]) => {
            if (active) {
                setOptions(results ?? []);
                setLoading(false)
            }
        });

        return () => {
            active = false;
        }
    }, [fetch, inputValue]);

    return (
        <Autocomplete
            size="small"
            value={value}
            onChange={(event: React.SyntheticEvent, value: string | null) => onSelected(value)}
            inputValue={inputValue}
            isOptionEqualToValue={(option: string, value: string | null) => option === value}
            onInputChange={(event: React.SyntheticEvent<Element, Event>, newValue: string) => newValue ? setInputValue(newValue) : setInputValue('')}
            options={options ?? []}
            loading={loading}
            renderInput={(params) =>
                <TextField
                    {...params}
                    placeholder="Version"
                    label="Version"
                    InputProps={{
                        ...params.InputProps,
                        endAdornment: (
                            <React.Fragment>
                                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                {params.InputProps.endAdornment}
                            </React.Fragment>
                        ),
                    }} />}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: string, { inputValue }) => {
                const matches = match(option, inputValue)
                const parts = parse(option, matches)
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
                            </Box>
                        </Box>
                    </Box>
                )
            }}
        />
    );
}

export default ModuleVersionAutocomplete
