import { Stack, Breadcrumbs } from '@mui/material';
import Link from '../routes/Link';

interface Route {
    title: string;
    path: string;
}

interface Props {
    childRoutes?: Route[] | null;
}

function AdminAreaBreadcrumbs({ childRoutes }: Props) {
    const childRoutePaths = childRoutes?.map(r => r.path) ?? [];

    return (
        <Stack direction="row" spacing={2} marginBottom={2}>
            <Breadcrumbs aria-label="admin breadcrumb">
                <Link color="inherit" to='/admin'>
                    admin area
                </Link>
                {childRoutes?.map(({ title, path }, i) =>
                    <Link key={path} color="inherit" to={`/admin/${childRoutePaths.slice(0, i + 1).join('/')}`}>{title}</Link>
                )}
            </Breadcrumbs>
        </Stack>
    );
}

export default AdminAreaBreadcrumbs
