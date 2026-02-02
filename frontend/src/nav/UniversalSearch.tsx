import {
    Autocomplete,
    AutocompleteRenderGroupParams,
    Avatar,
    Box,
    CircularProgress,
    Divider,
    TextField,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import teal from '@mui/material/colors/teal';
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useEffect, useState } from 'react';
import { fetchQuery, useRelayEnvironment } from 'react-relay';
import { useNavigate } from 'react-router-dom';
import { GroupIcon, TerraformIcon, WorkspaceIcon } from '../common/Icons';
import { UniversalSearchQuery, UniversalSearchQuery$data } from './__generated__/UniversalSearchQuery.graphql';

const searchQuery = graphql`
    query UniversalSearchQuery($query: String!) {
        search(query: $query) {
            results {
                __typename
                ... on NamespaceFavorite {
                    id
                    namespace {
                        __typename
                        ... on Group {
                            fullPath
                        }
                        ... on Workspace {
                            fullPath
                        }
                    }
                }
                ... on Group {
                    id
                    name
                    fullPath
                }
                ... on Workspace {
                    id
                    name
                    fullPath
                }
                ... on TerraformModule {
                    id
                    name
                    system
                    groupPath
                    registryNamespace
                }
                ... on TerraformProvider {
                    id
                    name
                    groupPath
                    registryNamespace
                }
                ... on Team {
                    id
                    name
                }
            }
        }
    }
`;

// Type definitions for search results
type GroupResult = {
    __typename: 'Group';
    id: string;
    name: string;
    fullPath: string;
};

type WorkspaceResult = {
    __typename: 'Workspace';
    id: string;
    name: string;
    fullPath: string;
};

type TeamResult = {
    __typename: 'Team';
    id: string;
    name: string;
};

type TerraformModuleResult = {
    __typename: 'TerraformModule';
    id: string;
    name: string;
    system: string;
    groupPath: string;
    registryNamespace: string;
};

type TerraformProviderResult = {
    __typename: 'TerraformProvider';
    id: string;
    name: string;
    groupPath: string;
    registryNamespace: string;
};

type NamespaceFavoriteResult = {
    __typename: 'NamespaceFavorite';
    id: string;
    namespace: {
        __typename: 'Group' | 'Workspace';
        fullPath: string;
    };
};

type SearchResult = GroupResult | WorkspaceResult | TeamResult | TerraformModuleResult | TerraformProviderResult | NamespaceFavoriteResult;

type AutocompleteOption = {
    category: string;
    id: string;
    label: string;
    icon: React.ReactNode;
    path: string;
}

// Icon generators
const ICON_GENERATORS = {
    NamespaceFavorite: (result: NamespaceFavoriteResult): React.ReactNode => {
        const type = result.namespace.__typename;
        return type === 'Group' ? <GroupIcon color="disabled" /> : <WorkspaceIcon color="disabled" />;
    },
    Group: (): React.ReactNode => <GroupIcon color="disabled" />,
    Workspace: (): React.ReactNode => <WorkspaceIcon color="disabled" />,
    Team: (result: TeamResult): React.ReactNode => <Avatar
        sx={{ width: 32, height: 32, bgcolor: teal[200] }}
        variant="rounded"
    >{result.name[0].toUpperCase()}</Avatar>,
    TerraformModule: (): React.ReactNode => <TerraformIcon color="disabled" />,
    TerraformProvider: (): React.ReactNode => <TerraformIcon color="disabled" />,
} as const;

// Label generators
const LABEL_GENERATORS = {
    NamespaceFavorite: (result: NamespaceFavoriteResult): string => result.namespace.fullPath,
    Group: (result: GroupResult): string => result.fullPath,
    Workspace: (result: WorkspaceResult): string => result.fullPath,
    Team: (result: TeamResult): string => result.name,
    TerraformModule: (result: TerraformModuleResult): string => `${result.groupPath}/${result.name}/${result.system}`,
    TerraformProvider: (result: TerraformProviderResult): string => `${result.groupPath}/${result.name}`,
} as const;

const PATH_GENERATORS = {
    NamespaceFavorite: (result: NamespaceFavoriteResult): string => `/groups/${result.namespace.fullPath}`,
    Group: (result: GroupResult): string => `/groups/${result.fullPath}`,
    Workspace: (result: WorkspaceResult): string => `/groups/${result.fullPath}`,
    TerraformModule: (result: TerraformModuleResult): string => `/module-registry/${result.registryNamespace}/${result.name}/${result.system}`,
    TerraformProvider: (result: TerraformProviderResult): string => `/provider-registry/${result.registryNamespace}/${result.name}`,
    Team: (result: TeamResult): string => `/teams/${result.name}`,
} as const;

const CATEGORIES = {
    NamespaceFavorite: 'Favorites',
    Group: 'Groups',
    Workspace: 'Workspaces',
    Team: 'Teams',
    TerraformModule: 'Terraform Modules',
    TerraformProvider: 'Terraform Providers',
} as const;

const mapResultsToOptions = (results: readonly any[]): AutocompleteOption[] => {
    return results.map((res) => {
        res = res as SearchResult;
        const type = res.__typename as keyof typeof CATEGORIES;
        return {
            category: CATEGORIES[type],
            id: res.id,
            label: LABEL_GENERATORS[type](res as any),
            path: PATH_GENERATORS[type](res as any),
            icon: ICON_GENERATORS[type](res as any),
        }
    });
};

function UniversalSearch() {
    const theme = useTheme();
    const navigate = useNavigate();
    const environment = useRelayEnvironment();
    const [options, setOptions] = useState<AutocompleteOption[] | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState<string>('');
    const [selectedItem, setSelectedItem] = useState<AutocompleteOption | null>(null);

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results: UniversalSearchQuery$data['search']) => void,
                ) => {
                    fetchQuery<UniversalSearchQuery>(
                        environment,
                        searchQuery,
                        { query: request.input }, { fetchPolicy: 'network-only' }
                    ).toPromise().then(async (response: UniversalSearchQuery$data | undefined) => {
                        if (response && response.search) {
                            callback(response.search);
                        }
                    });
                },
                500,
                { leading: false, trailing: true }
            ),
        [environment],
    );

    const search = () => {
        let active = true;

        setLoading(true);
        fetch({ input: inputValue }, (response: UniversalSearchQuery$data['search']) => {
            if (active) {
                setOptions(mapResultsToOptions(response.results));
                setLoading(false);
            }
        });

        return () => {
            active = false;
        };
    };

    useEffect(() => {
        return () => {
            // Cancel request when component unmounts
            fetch.cancel()
        }
    }, [fetch]);

    const handleFocus = () => {
        const callback = search();
        fetch.flush();

        return callback;
    };

    const handleBlur = () => {
        setOptions(null);
    };

    useEffect(() => {
        if (options === null) {
            return;
        }

        return search();
    }, [fetch, inputValue]);

    const onKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
        if (event.key === 'Enter') {
            fetch.flush();
        }
    };

    const onSelected = (item: AutocompleteOption | null) => {
        setInputValue('');
        setSelectedItem(null);
        if (item) {
            navigate(item.path);
        }
    };

    return (
        <Box sx={{
            width: '100%',
            maxWidth: 600,
            minWidth: 300,
            [theme.breakpoints.down('lg')]: {
                maxWidth: 400,
            },
            [theme.breakpoints.down('md')]: {
                display: 'none',
            }
        }}>
            <Autocomplete
                size="small"
                onKeyDown={onKeyDown}
                onChange={(_: React.SyntheticEvent, value: AutocompleteOption | null) => onSelected(value)}
                inputValue={inputValue}
                isOptionEqualToValue={(option: AutocompleteOption, value: AutocompleteOption) => option.id === value.id}
                onInputChange={(event: React.SyntheticEvent<Element, Event>, newValue: string) => newValue ? setInputValue(newValue) : setInputValue('')}
                options={options || []}
                loading={loading}
                getOptionLabel={(option: AutocompleteOption) => option.label}
                open={options !== null && (inputValue.length > 0 || options.length > 0)}
                value={selectedItem}
                blurOnSelect
                clearOnBlur
                clearOnEscape
                noOptionsText="No results found"
                groupBy={(option: AutocompleteOption) => option.category}
                renderGroup={((params: AutocompleteRenderGroupParams) => <Box key={params.key} mt={1}>
                    <Divider />
                    <Typography pt={1} pb={1} pl={2} pr={2} fontWeight={700} variant='body2'>{params.group}</Typography>
                    <Divider />
                    <Box>
                        {params.children}
                    </Box>
                </Box>)}
                renderInput={(params) =>
                    <TextField
                        {...params}
                        placeholder="Search"
                        onFocus={handleFocus}
                        onBlur={handleBlur}
                        InputProps={{
                            ...params.InputProps,
                            endAdornment: (
                                <React.Fragment>
                                    {loading ? <CircularProgress color="inherit" size={20} /> : null}
                                </React.Fragment>
                            ),
                        }} />}
                renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: AutocompleteOption, { inputValue }) => {
                    const label = option.label
                    const matches = match(label, inputValue)
                    const parts = parse(label, matches)
                    return (
                        <Box component="li" {...props}>
                            <Tooltip title={label}>
                                <Box width="100%" display="flex" justifyContent="space-between" alignItems="center">
                                    <Typography color="textSecondary" variant='body2' whiteSpace='nowrap' overflow='hidden' textOverflow='ellipsis' mr={2}>
                                        {parts.map((part: { text: string, highlight: boolean }, index: number) => (
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
                                    {option.icon}
                                </Box>
                            </Tooltip>
                        </Box>
                    )
                }}
            />
        </Box>
    );
}

export default UniversalSearch;
