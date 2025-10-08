import { Box, Typography } from "@mui/material";
import AdminAreaBreadcrumbs from "../AdminAreaBreadcrumbs";
import MaintenanceSettings from "./MaintenanceSettings";

const DESCRIPTION = 'Configure system-wide settings that affect the entire Tharsis platform, including maintenance mode and other operational controls.';

function SystemSettings() {
    return (
        <Box>
            <AdminAreaBreadcrumbs
                childRoutes={[
                    { title: "system settings", path: 'system_settings' }
                ]}
            />
            <Box>
                <Typography variant="h5" gutterBottom>System Settings</Typography>
                <Typography variant="body2" sx={{ mb: 3 }}>{DESCRIPTION}</Typography>
                <MaintenanceSettings />
            </Box>
        </Box>
    );
}

export default SystemSettings;
