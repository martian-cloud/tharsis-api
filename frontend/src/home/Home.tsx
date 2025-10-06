import React, { Suspense, useContext } from 'react';
import { Box, CircularProgress, Link, ListItemButton, Paper, ToggleButton, Typography, useTheme } from '@mui/material';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunchOutlined';
import HelpIcon from '@mui/icons-material/HelpOutline';
import graphql from 'babel-plugin-relay/macro';
import config from '../common/config';
import { useSearchParams } from 'react-router-dom';
import { PreloadedQuery, useFragment, usePreloadedQuery } from 'react-relay/hooks';
import { HomeQuery } from './__generated__/HomeQuery.graphql';
import { HomeFragment_activity$key } from './__generated__/HomeFragment_activity.graphql';
import { UserContext } from '../UserContext';
import HomeDrawer from './HomeDrawer';
import HomeActivityFeed from './HomeActivityFeed';
import HomeRunList from './HomeRunList';

const query = graphql`
    query HomeQuery {
      ...HomeFragment_activity
    }
`;

interface Props {
    queryRef: PreloadedQuery<HomeQuery>;
}

function Home(props: Props) {
    const theme = useTheme();
    const user = useContext(UserContext);
    const queryData = usePreloadedQuery<HomeQuery>(query, props.queryRef);
    const [searchParams, setSearchParams] = useSearchParams();

    // This filter will only show the current users activity when true
    const showMyActivity = searchParams.get('filter') === 'myActivity';

    const data = useFragment<HomeFragment_activity$key>(
        graphql`
      fragment HomeFragment_activity on Query {
        activityEvents(
            first: 0
        ) {
            totalCount
        }
        config {
            tharsisSupportUrl
        }
      }
    `, queryData);

    return (
        <Box display="flex">
            <HomeDrawer />
            <Box component="main" sx={{ flexGrow: 1 }}>
                <Box sx={{
                    maxWidth: 1400,
                    margin: 'auto',
                    display: 'flex',
                    [theme.breakpoints.down('lg')]: {
                        flexDirection: 'column-reverse',
                        alignItems: 'flex-start',
                        '& > *': { mb: 2 }
                    }
                }}>
                    {data.activityEvents.totalCount === 0 && <Box flex={1} mt={4} display="flex" justifyContent="center">
                        <Box
                            sx={{
                                p: 4,
                                maxWidth: 600,
                                display: 'flex',
                                flexDirection: 'column',
                                alignItems: 'center',
                                justifyContent: 'center',
                            }}>
                            <Typography variant="h6">Welcome to Tharsis!</Typography>
                            <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                                Get started using Tharsis to manage your Terraform deployments
                            </Typography>
                        </Box>
                    </Box>}
                    {data.activityEvents.totalCount > 0 && <React.Fragment>
                        <Box padding={`${theme.spacing(2)} ${theme.spacing(3)}`} flex={1}>
                            <Box display="flex" justifyContent="space-between" alignItems="center">
                                <Typography variant="h6" fontWeight={600} mb={1}>Activity</Typography>
                                <ToggleButton
                                    onChange={() => setSearchParams(!showMyActivity ? { filter: 'myActivity' } : {}, { replace: true })}
                                    color="secondary"
                                    selected={showMyActivity}
                                    size="small"
                                    value="myActivity">
                                    My Activity
                                </ToggleButton>
                            </Box>
                            <Suspense fallback={<Box
                                sx={{
                                    width: '100%',
                                    height: `calc(100vh - 64px)`,
                                    display: 'flex',
                                    alignItems: 'center',
                                    justifyContent: 'center'
                                }}>
                                <CircularProgress />
                            </Box>}>
                                <HomeActivityFeed username={showMyActivity ? user.username : undefined} />
                            </Suspense>
                        </Box>
                        <Box sx={{
                            padding: 2,
                            width: '100%',
                            [theme.breakpoints.up('lg')]: {
                                width: 400,
                                maxWidth: 400,
                            }
                        }}>
                            <Paper sx={{ mb: 3 }}>
                                <ListItemButton component={Link} target='_blank' rel='noopener noreferrer' href={config.docsUrl}>
                                    <RocketLaunchIcon sx={{ mr: 2 }} />
                                    <Box>
                                        <Typography variant="subtitle1" fontWeight={600}>Getting Started</Typography>
                                        <Typography variant="body2">Learn how to use Tharsis</Typography>
                                    </Box>
                                </ListItemButton>
                            </Paper>
                            {data.config.tharsisSupportUrl !== '' && <Paper sx={{ mb: 3 }}>
                                <ListItemButton component={Link} target='_blank' rel='noopener noreferrer' href={data.config.tharsisSupportUrl}>
                                    <HelpIcon sx={{ mr: 2 }} />
                                    <Box>
                                        <Typography variant="subtitle1" fontWeight={600}>Need assistance?</Typography>
                                        <Typography variant="body2">Contact our support team or report an issue</Typography>
                                    </Box>
                                </ListItemButton>
                            </Paper>}
                            <Paper sx={{ padding: 2 }}>
                                <HomeRunList />
                            </Paper>
                        </Box>
                    </React.Fragment>}
                </Box>
            </Box>
        </Box>
    );
}

export default Home;
