import { Box, CircularProgress, Typography } from "@mui/material";
import { Suspense } from "react";
import AdminAreaBreadcrumbs from "../AdminAreaBreadcrumbs";
import AdminAreaResourceLimitSettings from "./AdminAreaResourceLimitSettings";

const BREADCRUMB_ROUTES = [{ title: "resource limits", path: 'resource_limits' }];

function AdminAreaResourceLimitsPage() {
    return (
        <Suspense fallback={
            <Box>
                <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
                <Typography variant="h5" gutterBottom>Resource Limits</Typography>
                <Box display="flex" justifyContent="center" paddingTop={4}>
                    <CircularProgress size={24} />
                </Box>
            </Box>
        }>
            <Box>
                <AdminAreaBreadcrumbs childRoutes={BREADCRUMB_ROUTES} />
                <AdminAreaResourceLimitSettings />
            </Box>
        </Suspense>
    );
}

export default AdminAreaResourceLimitsPage;
