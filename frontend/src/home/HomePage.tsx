import HelpIcon from '@mui/icons-material/HelpOutline';
import RocketLaunchIcon from '@mui/icons-material/RocketLaunchOutlined';
import { Box, CircularProgress, Link, ListItemButton, Paper, Typography, useTheme } from '@mui/material';
import { Suspense, useContext } from 'react';
import { useAgentCopilot } from '../ai/AgentCopilotProvider';
import { ApiConfigContext } from '../ApiConfigContext';
import config from '../common/config';
import HomeActivityFeed from './HomeActivityFeed';
import HomeDrawer from './HomeDrawer';
import HomeRunList from './HomeRunList';

function HomePage() {
    const theme = useTheme();
    const apiConfig = useContext(ApiConfigContext);

    const { expanded: copilotExpanded } = useAgentCopilot();

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
                    <Box padding={`${theme.spacing(2)} ${theme.spacing(3)}`} flex={1}>
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
                            <HomeActivityFeed />
                        </Suspense>
                    </Box>
                    {!copilotExpanded && <Box sx={{
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
                        {apiConfig.tharsisSupportUrl !== '' && <Paper sx={{ mb: 3 }}>
                            <ListItemButton component={Link} target='_blank' rel='noopener noreferrer' href={apiConfig.tharsisSupportUrl}>
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
                    </Box>}
                </Box>
            </Box>
        </Box >
    );
}

export default HomePage;
