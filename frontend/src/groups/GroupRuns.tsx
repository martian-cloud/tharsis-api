import { Box, Typography } from '@mui/material';
import { Suspense } from 'react';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import NamespaceBreadcrumbs from '../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../skeletons/ListSkeleton';
import GroupRunList from './GroupRunList';
import { GroupRunsFragment_group$key } from './__generated__/GroupRunsFragment_group.graphql';

interface Props {
    fragmentRef: GroupRunsFragment_group$key;
}

const fragment = graphql`
    fragment GroupRunsFragment_group on Group {
        id
        fullPath
    }
`;

function GroupRuns({ fragmentRef }: Props) {
    const data = useFragment(fragment, fragmentRef);
    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[{ title: "runs", path: 'runs' }]}
            />
            <Box sx={{ mb: 1 }}>
                <Typography variant="h5" component="h1">
                    Runs
                </Typography>
            </Box>
            <Suspense fallback={<ListSkeleton rowCount={10} />}>
                <GroupRunList
                    groupPath={data.fullPath}
                    includeAssessmentRuns={false}
                />
            </Suspense>
        </Box>
    );
}

export default GroupRuns;
