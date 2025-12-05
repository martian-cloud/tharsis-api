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

type SearchResult = GroupResult | WorkspaceResult | TeamResult | TerraformModuleResult | TerraformProviderResult;

type AutocompleteOption = {
    category: string;
    id: string;
    label: string;
    icon: React.ReactNode;
    path: string;
}

// Icon generators
const ICON_GENERATORS = {
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
    Group: (result: GroupResult): string => result.fullPath,
    Workspace: (result: WorkspaceResult): string => result.fullPath,
    Team: (result: TeamResult): string => result.name,
    TerraformModule: (result: TerraformModuleResult): string => `${result.groupPath}/${result.name}/${result.system}`,
    TerraformProvider: (result: TerraformProviderResult): string => `${result.groupPath}/${result.name}`,
} as const;

const PATH_GENERATORS = {
    Group: (result: GroupResult): string => `/groups/${result.fullPath}`,
    Workspace: (result: WorkspaceResult): string => `/groups/${result.fullPath}`,
    TerraformModule: (result: TerraformModuleResult): string => `/module-registry/${result.registryNamespace}/${result.name}/${result.system}`,
    TerraformProvider: (result: TerraformProviderResult): string => `/provider-registry/${result.registryNamespace}/${result.name}`,
    Team: (result: TeamResult): string => `/teams/${result.name}`,
} as const;

const CATEGORIES = {
    Group: 'Groups',
    Workspace: 'Workspaces',
    Team: 'Teams',
    TerraformModule: 'Terraform Modules',
    TerraformProvider: 'Terraform Providers',
} as const;

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

    useEffect(() => {
        return () => {
            // Cancel request when component unmounts
            fetch.cancel()
        }
    }, [fetch]);

    useEffect(() => {
        let active = true;

        if (inputValue === '') {
            setOptions(null);
            setLoading(false);
        } else {
            setLoading(true);

            fetch({ input: inputValue }, (response: UniversalSearchQuery$data['search']) => {
                if (active) {
                    setOptions(response.results.map((res) => {
                        res = res as SearchResult;

                        const type = res.__typename;
                        const labelGenerator = LABEL_GENERATORS[type];
                        const iconGenerator = ICON_GENERATORS[type];
                        const pathGenerator = PATH_GENERATORS[type];
                        return {
                            category: CATEGORIES[type],
                            id: res.id,
                            label: labelGenerator(res as any),
                            path: pathGenerator(res as any),
                            icon: iconGenerator(res as any),
                        }
                    }));
                    setLoading(false);
                }
            });
        }

        return () => {
            active = false;
        };
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
                open={options !== null}
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
