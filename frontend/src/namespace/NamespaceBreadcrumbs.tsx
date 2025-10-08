import { IconButton, Stack, Breadcrumbs } from '@mui/material';
import CopyIcon from '@mui/icons-material/ContentCopy';
import React from 'react';
import Link from '../routes/Link';

interface Route {
    title: string;
    path: string;
}

interface Props {
    namespacePath: string;
    childRoutes?: Route[] | null;
}

function NamespaceBreadcrumbs(props: Props) {
    const pathParts = props.namespacePath.split("/") ?? [];
    const childRoutePaths = props.childRoutes?.map(r => r.path) ?? [];

    return (
        <Stack direction="row" spacing={2} marginBottom={2}>
            <Breadcrumbs aria-label="group breadcrumb">
                <Link color="inherit" to={'/groups'}>
                    groups
                </Link>
                {pathParts.map((name, i) => (
                    <Link key={name} color="inherit" to={`/groups/${pathParts.slice(0, i + 1).join('/')}`}>
                        {name}
                    </Link>
                ))}
                {props?.childRoutes?.map(({ title, path }, i) => (
                    path[0] === '/' ?
                        <Link key={path} color="inherit" to={path}>{title}</Link>
                        :
                        <Link key={path} color="inherit" to={`/groups/${props.namespacePath}/-/${childRoutePaths.slice(0, i + 1).join('/')}`}>{title}</Link>
                ))}
            </Breadcrumbs>
            {childRoutePaths.length == 0 &&
                <IconButton
                    sx={{
                        padding: '0px',
                        opacity: '20%',
                        transition: 'ease',
                        transitionDuration: '300ms',
                        ":hover": {
                            opacity: '100%'
                        }
                    }}
                    onClick={() => navigator.clipboard.writeText(props.namespacePath)}
                    title="Copy Path"
                >
                    <CopyIcon sx={{ width: 16, height: 16 }} />
                </IconButton>
            }
        </Stack>
    );
}

export default NamespaceBreadcrumbs;
