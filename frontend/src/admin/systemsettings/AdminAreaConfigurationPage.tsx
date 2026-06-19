import { Box, CircularProgress, Typography } from "@mui/material";
import { Suspense } from "react";
import AdminAreaBreadcrumbs from "../AdminAreaBreadcrumbs";
import AdminAreaConfigurationSettings from "./AdminAreaConfigurationSettings";

const BREADCRUMB_ROUTES = [{ title: "configuration", path: 'configuration' }];

function AdminAreaConfigurationPage() {
    return (
        <Suspense fallback={
            <Box>
                <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
                <Typography variant="h5" gutterBottom>API Configuration</Typography>
                <Box display="flex" justifyContent="center" paddingTop={4}>
                    <CircularProgress size={24} />
                </Box>
            </Box>
        }>
            <Box>
                <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
                <AdminAreaConfigurationSettings />
            </Box>
        </Suspense>
    );
}

export default AdminAreaConfigurationPage;
