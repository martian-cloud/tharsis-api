import { Typography, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink, Route, Routes } from 'react-router-dom';
import SearchInput from '../../common/SearchInput';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import NamespaceMembershipList from './NamespaceMembershipList';
import NewNamespaceMembership from './NewNamespaceMembership';
import { NamespaceMembershipsFragment_memberships$key } from './__generated__/NamespaceMembershipsFragment_memberships.graphql';
import { NamespaceMembershipsIndexFragment_memberships$key } from './__generated__/NamespaceMembershipsIndexFragment_memberships.graphql';

interface Props {
    fragmentRef: NamespaceMembershipsFragment_memberships$key
}

function NamespaceMemberships(props: Props) {
    const data = useFragment<NamespaceMembershipsFragment_memberships$key>(
        graphql`
        fragment NamespaceMembershipsFragment_memberships on Namespace
        {
            ...NamespaceMembershipsIndexFragment_memberships
            ...NewNamespaceMembershipFragment_memberships
        }
      `, props.fragmentRef);
    return (
        <Box>
            <Routes>
                <Route index element={<NamespaceMembershipsIndex fragmentRef={data} />} />
                <Route path={`new`} element={<NewNamespaceMembership fragmentRef={data} />} />
            </Routes>
        </Box>
    );
}

interface NamespaceMembershipsIndexProps {
    fragmentRef: NamespaceMembershipsIndexFragment_memberships$key
}

function NamespaceMembershipsIndex(props: NamespaceMembershipsIndexProps) {
    const theme = useTheme();

    const data = useFragment<NamespaceMembershipsIndexFragment_memberships$key>(
        graphql`
        fragment NamespaceMembershipsIndexFragment_memberships on Namespace
        {
            fullPath
            ...NamespaceMembershipListFragment_memberships
        }
      `, props.fragmentRef);

    const [search, setSearch] = useState('');

    const onSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(event.target.value.toLowerCase());
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[
                    { title: "members", path: 'members' }
                ]}
            />
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                justifyContent: 'space-between',
                marginBottom: 2,
                [theme.breakpoints.down('md')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *:not(:last-child)': { marginBottom: 2 },
                }
            }}>
                <Box>
                    <Typography variant="h5" gutterBottom>Members</Typography>
                    <Typography variant="body2">
                        A member is associated with a role and can be a user, team, or service account
                    </Typography>
                </Box>
                <Box>
                    <Button component={RouterLink} variant="outlined" color="primary" to="new">Add Member</Button>
                </Box>
            </Box>
            <SearchInput
                placeholder="search for members"
                fullWidth
                sx={{ marginBottom: 2 }}
                onChange={onSearchChange}
            />
            <NamespaceMembershipList fragmentRef={data} search={search} />
        </Box>
    );
}

export default NamespaceMemberships;
