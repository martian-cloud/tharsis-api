import React, { Suspense } from 'react';
import { Avatar, Box, CircularProgress, Stack, Typography, useTheme } from '@mui/material';
import { teal } from '@mui/material/colors';
import { PreloadedQuery, useFragment, usePreloadedQuery } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import TeamMemberList from './TeamMemberList';
import { TeamDetailsQuery } from './__generated__/TeamDetailsQuery.graphql';
import { TeamDetailsFragment_team$key } from './__generated__/TeamDetailsFragment_team.graphql';
import TRNButton from '../common/TRNButton';

const query = graphql`
    query TeamDetailsQuery($name: String!, $first: Int!, $after: String) {
        team(name: $name) {
            ...TeamDetailsFragment_team
        }
    }
`;

interface Props {
    queryRef: PreloadedQuery<TeamDetailsQuery>;
}

function TeamDetails({ queryRef }: Props) {
    const theme = useTheme();
    const queryData = usePreloadedQuery<TeamDetailsQuery>(query, queryRef);

    const data = useFragment<TeamDetailsFragment_team$key>(
        graphql`
        fragment TeamDetailsFragment_team on Team
        {
            name
            description
            metadata {
                trn
            }
            ...TeamMemberListFragment_members
        }
        `,
        queryData.team
    );

    if (data) {
        return (
            <Box maxWidth={1000} margin="auto" padding={2}>
                <Suspense fallback={<Box
                    sx={{
                        width: '100%',
                        height: `calc(100vh - 64px)`,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}>
                    <CircularProgress />
                </Box>}
                >
                    <Box sx={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        [theme.breakpoints.down('sm')]: {
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            '& > *': { marginBottom: 2 },
                        }
                    }}>
                        <Box display="flex" marginBottom={4} alignItems="center">
                            <Avatar
                                sx={{
                                    width: 56,
                                    height: 56,
                                    marginRight: 2,
                                    bgcolor: teal[200]
                                }}
                                variant="rounded">{data.name[0].toUpperCase()}
                            </Avatar>
                            <Stack>
                                <Typography noWrap variant="h5" sx={{ maxWidth: 400, fontWeight: "bold" }}>{data.name}</Typography>
                                <Typography
                                    color="textSecondary"
                                    variant="subtitle2">{`${data.description.slice(0, 50)}${data.description.length > 50 ? '...' : ''}`}
                                </Typography>
                            </Stack>
                        </Box>
                        <TRNButton trn={data.metadata.trn} />
                    </Box>
                    <TeamMemberList fragmentRef={data} />
                </Suspense>
            </Box>
        );
    } else {
        return <Box display="flex" justifyContent="center" paddingTop={2}>
            <Typography color="textSecondary">Team not found</Typography>
        </Box>;
    }
}

export default TeamDetails;
