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
            <Box padding={2}>
                <Box mb={4}>
                    <Typography mb={1} variant="subtitle1" fontWeight={600}>Workspaces</Typography>
                    <HomeWorkspaceList />
                </Box>
                <Box>
                    <Typography mb={1} variant="subtitle1" fontWeight={600}>Teams</Typography>
                    <HomeTeamList />
                </Box>
            </Box>
        </Drawer>
    );
}

export default HomeDrawer;
