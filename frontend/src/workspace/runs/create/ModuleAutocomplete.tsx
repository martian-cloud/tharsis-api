import React, { useCallback, useEffect, useState } from 'react';
import { Autocomplete, Box, CircularProgress, TextField, Typography } from "@mui/material";
import { useRelayEnvironment } from "react-relay/hooks";
import { fetchQuery } from 'relay-runtime';
import throttle from 'lodash.throttle';
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import graphql from 'babel-plugin-relay/macro';
import { ModuleAutocompleteQuery } from './__generated__/ModuleAutocompleteQuery.graphql';
import { TerraformIcon } from '../../../common/Icons';

export interface ModuleOption {
    readonly id: string,
    readonly label: string
    readonly source: string,
    readonly private: boolean,
    readonly resourcePath: string,
    readonly groupPath: string,
    readonly registryNamespace: string
    readonly name: string,
    readonly system: string
}

interface Props {
    workspacePath: string
    onSelected: (value: ModuleOption | null) => void
}

function ModuleAutocomplete({ onSelected, workspacePath }: Props) {
    const [options, setOptions] = useState<any>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState<string>('');

    const filterModules = useCallback(((options: any) => {
        return options.filter((op: ModuleOption) => (!op.private || workspacePath.startsWith(`${op.groupPath}/`)))
    }), [workspacePath])

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: any) => void,
                ) => {
                    fetchQuery<ModuleAutocompleteQuery>(
                        environment,
                        graphql`
                            query ModuleAutocompleteQuery($search: String!) {
                                terraformModules(first: 50, search: $search) {
                                    edges {
                                        node {
                                            id
                                            name
                                            source
                                            private
                                            resourcePath
                                            groupPath
                                            registryNamespace
                                            name
                                            system
                                        }
                                    }
                                }

                            }
                            `,
                        { search: request.input }, { fetchPolicy: 'network-only' }
                    ).toPromise().then(async (response: any) => {
                        const options = response?.terraformModules.edges.map((edge: { node: any }) => {
                            return {
                                id: edge.node.id,
                                label: `${edge.node.registryNamespace}/${edge.node.name}/${edge.node.system}`,
                                source: edge.node.source,
                                private: edge.node.private ? true : false,
                                resourcePath: edge.node.resourcePath,
                                groupPath: edge.node.groupPath,
                                registryNamespace: edge.node.registryNamespace,
                                name: edge.node.name,
                                system: edge.node.system
                            }
                        })
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

        fetch({ input: inputValue }, (results?: ModuleOption[]) => {
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
            onChange={(event: React.SyntheticEvent, value: ModuleOption | null) => onSelected(value)}
            inputValue={inputValue}
            isOptionEqualToValue={(option: ModuleOption, value: ModuleOption) => option.id === value.id}
            onInputChange={(event: React.SyntheticEvent<Element, Event>, newValue: string) => newValue ? setInputValue(newValue) : setInputValue('')}
            options={options ?? []}
            filterOptions={filterModules}
            loading={loading}
            renderInput={(params) =>
                <TextField
                    {...params}
                    placeholder="Module"
                    label="Module"
                    InputProps={{
                        ...params.InputProps,
                        endAdornment: (
                            <React.Fragment>
                                {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                {params.InputProps.endAdornment}
                            </React.Fragment>
                        ),
                    }} />}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: any, { inputValue }) => {
                const matches = match(option.label, inputValue)
                const parts = parse(option.label, matches)
                return (
                    <Box component="li" {...props}>
                        <Box width="100%" display="flex" justifyContent="space-between" alignItems="center">
                            <Typography>
                                {parts.map((part: { text: string, highlight: boolean }, index: number) => (
                                    <span
                                        key={index}
                                    >
                                        {part.text}
                                    </span>
                                ))}
                            </Typography>
                            <TerraformIcon color="disabled" />
                        </Box>
                    </Box>
                )
            }}
        />
    );
}

export default ModuleAutocomplete
