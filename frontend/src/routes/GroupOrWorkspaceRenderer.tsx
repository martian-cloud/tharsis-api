import { Typography } from '@mui/material';
import { Box } from '@mui/system';
import graphql from 'babel-plugin-relay/macro';
import { PreloadedQuery, usePreloadedQuery } from 'react-relay/hooks';
import GroupDetails from '../groups/GroupDetails';
import WorkspaceDetails from '../workspace/WorkspaceDetails';
import { GroupOrWorkspaceRendererQuery } from './__generated__/GroupOrWorkspaceRendererQuery.graphql';

const query = graphql`
    query GroupOrWorkspaceRendererQuery($fullPath: String!) {
      namespace(fullPath: $fullPath) {
        __typename
        id
        fullPath
        ...on Group {
          ...GroupDetailsFragment_group
        }
        ...on Workspace {
          ...WorkspaceDetailsFragment_workspace
        }
      }
    }
`;

interface Props {
  queryRef: PreloadedQuery<GroupOrWorkspaceRendererQuery>
  route: string
}

function GroupOrWorkspaceDetails(props: Props) {
  const queryData = usePreloadedQuery<GroupOrWorkspaceRendererQuery>(query, props.queryRef);

  if (queryData.namespace && queryData.namespace.__typename === 'Group') {
    return <GroupDetails fragmentRef={queryData.namespace} route={props.route} />;
  } else if (queryData.namespace && queryData.namespace.__typename === 'Workspace') {
    return <WorkspaceDetails fragmentRef={queryData.namespace} route={props.route} />;
  } else {
    return (
      <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" height="400px">
        <Typography variant="h3" color="textSecondary">404</Typography>
        <Typography variant="h5" color="textSecondary">The page you're looking for does not exist or you're not authorized to view it</Typography>
      </Box>
    );
  }
}

export default GroupOrWorkspaceDetails;
