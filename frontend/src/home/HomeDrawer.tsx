import { Box, Typography } from '@mui/material';
import Drawer from '../common/Drawer';
import HomeWorkspaceList from './HomeWorkspaceList';
import HomeTeamList from './HomeTeamList';

const DRAWER_WIDTH = 350;

function HomeDrawer() {
    return (
        <Drawer
            width={DRAWER_WIDTH}
            mobileWidth={0}
            variant="permanent"
        >
            <Box sx={{
                padding: 2
            }}>
                <Box sx={{
                    mb: 4
                }}>
                    <Typography
                        variant="subtitle1"
                        sx={{
                            mb: 1,
                            fontWeight: 600
                        }}>Workspaces</Typography>
                    <HomeWorkspaceList />
                </Box>
                <Box>
                    <Typography
                        variant="subtitle1"
                        sx={{
                            mb: 1,
                            fontWeight: 600
                        }}>Teams</Typography>
                    <HomeTeamList />
                </Box>
            </Box>
        </Drawer>
    );
}

export default HomeDrawer;
