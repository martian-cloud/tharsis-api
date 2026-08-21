import { Autocomplete, Box, Chip, ListItem, ListItemText, TextField } from '@mui/material';
import CircularProgress from '@mui/material/CircularProgress';
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useEffect, useState } from 'react';
import { useRelayEnvironment } from "react-relay/hooks";
import { fetchQuery } from 'relay-runtime';
import { PackageAutocompleteQuery } from './__generated__/PackageAutocompleteQuery.graphql';

export interface PackageOption {
    readonly id: string;
    readonly label: string;
    readonly packageSource: string;
    readonly visibility: string;
    // The package's own description, shown under its source in the dropdown. Empty for a package that
    // has none, and for a source that resolves to no package at all.
    readonly description: string;
}

interface Props {
    groupPath: string;
    onSelected: (value: PackageOption | null) => void;
    filterOptions: (options: PackageOption[]) => PackageOption[];
    /**
     * The current selection. Always controlled: the caller owns the value that gets submitted, so
     * the field has to render that state rather than a private copy of it. Callers that collapse to
     * their own rendering once a package is picked simply pass null while the field is mounted.
     */
    value: PackageOption | null;
}

// freeFormPackage stands in for a source the registry has no package for, which is allowed: a policy
// may name a package that has not been published yet, and nothing on the backend requires it to
// exist. Only packageSource is ever submitted, so the empty id, visibility and description are
// display-only gaps.
function freeFormPackage(source: string): PackageOption {
    return {
        id: '',
        label: source,
        packageSource: source,
        visibility: '',
        description: '',
    };
}

// freeSolo widens the option type of every callback below to `string | PackageOption`, even though the
// options array itself only ever holds resolved packages. These two narrow at that boundary.
function isPackageOption(option: PackageOption | string): option is PackageOption {
    return typeof option !== 'string';
}

function optionLabel(option: PackageOption | string): string {
    return isPackageOption(option) ? option.label : option;
}

// PackageAutocomplete picks the package a policy evaluates. Options are every package visible to the
// group — its own and its ancestors', plus root-group and global ones owned anywhere else — and each
// is labelled by its fully-qualified source, since a bare package name is unique only within a group.
// The field also accepts a typed source that matches no option at all.
function PackageAutocomplete(props: Props) {
    const { groupPath, onSelected, filterOptions, value } = props;

    const [options, setOptions] = useState<ReadonlyArray<PackageOption> | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState(value?.label ?? '');

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly PackageOption[]) => void,
                ) => {
                    fetchQuery<PackageAutocompleteQuery>(
                        environment,
                        graphql`
                          query PackageAutocompleteQuery($first: Int, $fullPath: String!, $search: String!) {
                            group(fullPath: $fullPath) {
                                visiblePackages(first: $first, search: $search, sort: GROUP_LEVEL_DESC) {
                                    edges {
                                        node {
                                            id
                                            name
                                            description
                                            groupPath
                                            visibility
                                        }
                                    }
                                }
                            }
                          }
                        `,
                        // Deepest owning group first, so the group's own and nearest-ancestor packages
                        // come before root-level global ones. The visible set spans every root group,
                        // so first also has to be generous enough to be useful before anything is typed.
                        { search: request.input, first: 50, fullPath: groupPath },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(response => {
                        const opts: PackageOption[] = [];
                        for (const edge of response?.group?.visiblePackages?.edges ?? []) {
                            const node = edge?.node;
                            // Skip edges whose node is null (e.g. a stale/soft-deleted package
                            // reference): without a real id/visibility the option would render
                            // "undefined" and break isOptionEqualToValue. Better to omit it.
                            if (!node) {
                                continue;
                            }
                            const packageSource = node.groupPath && node.name
                                ? `${node.groupPath}/${node.name}` : '';
                            opts.push({
                                id: node.id,
                                label: packageSource,
                                packageSource,
                                visibility: node.visibility,
                                description: node.description ?? '',
                            });
                        }
                        callback(opts);
                    });
                },
                500,
            ),
        [environment, groupPath],
    );

    useEffect(() => {
        let active = true;
        setLoading(true);
        fetch({ input: inputValue }, (results?: readonly PackageOption[]) => {
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
            size="small"
            // freeSolo because the package need not exist: a policy can be authored before its
            // package is published, and the backend accepts any source string.
            freeSolo
            inputValue={inputValue}
            value={value}
            onChange={(_event: React.SyntheticEvent, newValue: PackageOption | string | null) => {
                // freeSolo hands back a bare string when the user commits text that matched no option.
                const selected = typeof newValue === 'string' ? freeFormPackage(newValue) : newValue;
                setInputValue(selected?.label ?? '');
                onSelected(selected);
            }}
            onInputChange={(_, newInputValue: string, reason) => {
                setInputValue(newInputValue);
                // Typed text is the package source, so capture it on every keystroke rather than
                // waiting for Enter — there is no commit step to forget. 'reset' is skipped because it
                // fires when an option is picked, and that option's id and visibility must survive.
                if (reason === 'input') {
                    onSelected(newInputValue ? freeFormPackage(newInputValue) : null);
                } else if (reason === 'clear') {
                    onSelected(null);
                }
            }}
            filterOptions={(options) => filterOptions(options.filter(isPackageOption))}
            // A policy stores only its package source, so compare on that and fall back to the id
            // for callers that do have a resolved package.
            isOptionEqualToValue={(option, value) => {
                if (!isPackageOption(option) || !isPackageOption(value)) {
                    return optionLabel(option) === optionLabel(value);
                }
                return option.packageSource && value.packageSource
                    ? option.packageSource === value.packageSource
                    : option.id === value.id;
            }}
            getOptionLabel={optionLabel}
            renderOption={(renderProps: React.HTMLAttributes<HTMLLIElement> & { key?: React.Key }, option: PackageOption | string, { inputValue }) => {
                const { key, ...optionProps } = renderProps;
                const label = optionLabel(option);
                const matches = match(label, inputValue);
                const parts = parse(label, matches);
                const description = isPackageOption(option) ? option.description : '';
                return (
                    <ListItem dense key={key} {...optionProps}>
                        <ListItemText
                            primary={
                                parts.map((part: any, index: number) => (
                                    <span
                                        key={index}
                                        style={{ fontWeight: part.highlight ? 700 : 400 }}
                                    >
                                        {part.text}
                                    </span>
                                ))}
                            primaryTypographyProps={{ noWrap: true }}
                            // undefined rather than an empty string so a package without a description
                            // renders as a single-line option instead of a blank second line.
                            secondary={description || undefined}
                            secondaryTypographyProps={{ noWrap: true }}
                            // minWidth lets both lines clip to the row instead of widening it past the
                            // visibility chip, which is what makes noWrap take effect.
                            sx={{ minWidth: 0, mr: 1 }}
                        />
                        <Box flex={1} />
                        {isPackageOption(option) && <Chip size="small" variant="outlined" label={option.visibility} />}
                    </ListItem>
                );
            }}
            options={options ?? []}
            loading={loading}
            renderInput={(params) => (
                <TextField
                    {...params}
                    placeholder='Select or enter a package source'
                    slotProps={{
                        input: {
                            ...params.InputProps,
                            endAdornment: (
                                <React.Fragment>
                                    {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                    {params.InputProps.endAdornment}
                                </React.Fragment>
                            ),
                        }
                    }}
                />
            )}
        />
    );
}

export default PackageAutocomplete;
