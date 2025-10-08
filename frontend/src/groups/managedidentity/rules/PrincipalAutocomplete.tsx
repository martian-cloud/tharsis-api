import TeamIcon from '@mui/icons-material/PeopleOutline';
import UserIcon from '@mui/icons-material/PersonOutline';
import { Autocomplete, Avatar, Box, ListItem, ListItemText, styled, TextField, useTheme } from '@mui/material';
import CircularProgress from '@mui/material/CircularProgress';
import teal from '@mui/material/colors/teal';
import match from 'autosuggest-highlight/match';
import parse from 'autosuggest-highlight/parse';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import { LanConnect as ServiceAccountIcon } from 'mdi-material-ui';
import React, { useEffect, useState } from 'react';
import { useRelayEnvironment } from "react-relay/hooks";
import { fetchQuery } from 'relay-runtime';
import Gravatar from '../../../common/Gravatar';
import { PrincipalAutocompleteQuery } from './__generated__/PrincipalAutocompleteQuery.graphql';

export interface TeamOption {
    readonly type: string;
    readonly id: string;
    readonly label: string;
    readonly name: string;
    readonly avatarLabel: string;
}

export interface ServiceAccountOption {
    readonly type: string;
    readonly id: string;
    readonly label: string;
    readonly name: string;
    readonly resourcePath: string;
    readonly avatarLabel: string;
}

export interface UserOption {
    readonly type: string;
    readonly id: string;
    readonly label: string;
    readonly username: string;
    readonly email: string
    readonly avatarLabel: string;
}

export type Option = UserOption | ServiceAccountOption | TeamOption;

interface Props {
    groupPath: string;
    onSelected: (value: Option | null) => void;
    filterOptions: (options: Option[]) => Option[];
}

const StyledAvatar = styled(
    Avatar
)(() => ({
    width: 24,
    height: 24,
    backgroundColor: teal[200],
}))

function PrincipalAutocomplete(props: Props) {
    const { groupPath, onSelected, filterOptions } = props;

    const theme = useTheme();
    const [options, setOptions] = useState<ReadonlyArray<Option> | null>(null);
    const [loading, setLoading] = useState<boolean>(false);
    const [inputValue, setInputValue] = useState('');

    const environment = useRelayEnvironment();

    const fetch = React.useMemo(
        () =>
            throttle(
                (
                    request: { input: string },
                    callback: (results?: readonly Option[]) => void,
                ) => {
                    fetchQuery<PrincipalAutocompleteQuery>(
                        environment,
                        graphql`
                          query PrincipalAutocompleteQuery($first: Int, $fullPath: String!, $search: String!) {
                            group(fullPath: $fullPath){
                                serviceAccounts(first: $first, search: $search, includeInherited: true) {
                                    edges {
                                        node {
                                            id
                                            name
                                            resourcePath
                                        }
                                    }
                                }
                            }
                            teams (first: $first, search: $search){
                                edges {
                                    node {
                                        id
                                        name
                                    }
                                }
                            }
                            users (first: $first, search: $search) {
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
                        { search: request.input, first: 20, fullPath: groupPath },
                        { fetchPolicy: 'network-only' }
                    ).toPromise().then(async response => {
                        const options = [
                            ...response?.teams?.edges?.map(edge => ({
                                type: 'team',
                                id: edge?.node?.id,
                                label: edge?.node?.name,
                                name: edge?.node?.name,
                                avatarLabel: edge?.node?.name[0].toUpperCase()
                            } as TeamOption)) || [],
                            ...response?.users?.edges?.map(edge => ({
                                type: 'user',
                                id: edge?.node?.id,
                                label: edge?.node?.username,
                                username: edge?.node?.username,
                                email: edge?.node?.email,
                                avatarLabel: edge?.node?.email
                            } as UserOption)) || [],
                            ...response?.group?.serviceAccounts?.edges?.map(edge => ({
                                type: 'serviceaccount',
                                id: edge?.node?.id,
                                label: edge?.node?.resourcePath,
                                name: edge?.node?.name,
                                resourcePath: edge?.node?.resourcePath,
                                avatarLabel: edge?.node?.name[0].toUpperCase()
                            } as ServiceAccountOption)) || []
                        ];
                        callback(options);
                    });
                },
                500,
            ),
        [environment],
    );

    useEffect(() => {
        let active = true;

        setLoading(true);

        fetch({ input: inputValue }, (results?: readonly Option[]) => {
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
            inputValue={inputValue}
            value={null}
            onChange={(event: React.SyntheticEvent, value: Option | null) => {
                setInputValue('');
                onSelected(value);
            }}
            onInputChange={(_, newInputValue: string) => setInputValue(newInputValue)}
            filterOptions={filterOptions}
            isOptionEqualToValue={(option: Option, value: Option) => option.id === value.id}
            getOptionLabel={(option: Option) => option.label}
            renderOption={(props: React.HTMLAttributes<HTMLLIElement>, option: Option, { inputValue }) => {
                const matches = match(option.label, inputValue);
                const parts = parse(option.label, matches);
                return (
                    <ListItem dense {...props}>
                        <Box marginRight={1}>
                            {option.type === 'user' && <Gravatar width={24} height={24} email={option.avatarLabel} />}
                            {(option.type === 'team' || option.type === 'serviceaccount') && <StyledAvatar>{option.avatarLabel}</StyledAvatar>}
                        </Box>
                        <ListItemText primary={
                            parts.map((part: any, index: number) => (
                                <span
                                    key={index}
                                    style={{
                                        fontWeight: part.highlight ? 700 : 400,
                                    }}
                                >
                                    {part.text}
                                </span>
                            ))}
                            primaryTypographyProps={{ noWrap: true }}
                        />
                        <Box flex={1} />
                        <Box marginLeft={1}>
                            {option.type === 'user' && <UserIcon sx={{ color: theme.palette.text.secondary }} />}
                            {option.type === 'team' && <TeamIcon sx={{ color: theme.palette.text.secondary }} />}
                            {option.type === 'serviceaccount' && <ServiceAccountIcon sx={{ color: theme.palette.text.secondary }} />}
                        </Box>
                    </ListItem>
                );
            }}
            options={options ?? []}
            loading={loading}
            renderInput={(params) => (
                <TextField
                    {...params}
                    placeholder='Select a principal to add'
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

export default PrincipalAutocomplete;
