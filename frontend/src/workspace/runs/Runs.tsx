import { Suspense, useState } from 'react';
import { Box, Button, Checkbox, CircularProgress, Menu, MenuItem, Stack } from '@mui/material';
import FilterListIcon from '@mui/icons-material/FilterList';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Link as RouterLink } from 'react-router-dom';
import { Route, Routes } from 'react-router-dom';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import RunDetails from './RunDetails';
import RunList from './RunList';
import CreateRun from './create/CreateRun';
import { RunsFragment_runs$key } from './__generated__/RunsFragment_runs.graphql';
import { RunsIndexFragment_runs$key } from './__generated__/RunsIndexFragment_runs.graphql';

interface Props {
    fragmentRef: RunsFragment_runs$key
}

function Runs({ fragmentRef }: Props) {
    const data = useFragment(
        graphql`
        fragment RunsFragment_runs on Workspace
        {
            fullPath
            ...RunsIndexFragment_runs
            ...CreateRunFragment_workspace
            ...RunDetailsFragment_details
        }
      `, fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<RunsIndex fragmentRef={data} />} />
                <Route path={`create`} element={<CreateRun fragmentRef={data} />} />
                <Route path={`:id/*`} element={<RunDetails fragmentRef={data} />} />
            </Routes>
        </Box>
    );
}

interface RunsIndexProps {
    fragmentRef: RunsIndexFragment_runs$key
}

function RunsIndex({ fragmentRef }: RunsIndexProps) {

    const data = useFragment<RunsIndexFragment_runs$key>(
        graphql`
        fragment RunsIndexFragment_runs on Workspace
        {
            id
            fullPath
        }
      `, fragmentRef);

    const [menuAnchorEl, setMenuAnchorEl] = useState<null | HTMLElement>(null);
    const [showAssessmentRuns, setShowAssessmentRuns] = useState<boolean>(false);

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[{ title: "runs", path: 'runs' }]}
            />
            <Box
                sx={{
                    mb: 2,
                    display: 'flex',
                    flexDirection: 'row',
                    justifyContent: 'space-between'
                }}>
                <Typography variant="h5">Runs</Typography>
                <Stack direction="row" spacing={1}>
                    <Button
                        component={RouterLink}
                        variant="outlined"
                        color="primary"
                        to="create">
                        Create Run
                    </Button>
                    <Button
                        id="filter-button"
                        color="info"
                        variant="outlined"
                        aria-controls={menuAnchorEl ? 'basic-menu' : undefined}
                        aria-haspopup="true"
                        aria-expanded={menuAnchorEl ? 'true' : undefined}
                        onClick={(event: React.MouseEvent<HTMLButtonElement>) => setMenuAnchorEl(event.currentTarget)}
                    >
                        <FilterListIcon />
                    </Button>
                    <Menu
                        id="basic-menu"
                        anchorEl={menuAnchorEl}
                        open={Boolean(menuAnchorEl)}
                        onClose={() => setMenuAnchorEl(null)}
                        disableScrollLock
                        MenuListProps={{
                            'aria-labelledby': 'filter-button',
                        }}
                    >
                        <MenuItem
                            onClick={() => setShowAssessmentRuns(!showAssessmentRuns)}
                        >
                            <Checkbox
                                color="info"
                                checked={showAssessmentRuns}
                            />
                            Show Assessment Runs
                        </MenuItem>
                    </Menu>
                </Stack>
            </Box>
            <Suspense fallback={<Box
                sx={{
                    width: '100%',
                    height: `calc(100vh - 64px)`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center'
                }}
            >
                <CircularProgress />
            </Box>}>
                <RunList
                    workspaceId={data.id}
                    workspacePath={data.fullPath}
                    includeAssessmentRuns={showAssessmentRuns}
                />
            </Suspense>
        </Box>
    );
}

export default Runs;
