import { Box, Typography } from "@mui/material";
import AdminAreaBreadcrumbs from "../AdminAreaBreadcrumbs";
import AdminAreaMaintenanceSettings from "./AdminAreaMaintenanceSettings";

const DESCRIPTION = 'Configure system-wide settings that affect the entire Tharsis platform, including maintenance mode and other operational controls.';

function AdminAreaSystemSettings() {
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
                <AdminAreaMaintenanceSettings />
            </Box>
        </Box>
    );
}

export default AdminAreaSystemSettings;
