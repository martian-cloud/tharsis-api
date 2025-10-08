import { Paper, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { ErrorBoundary } from 'react-error-boundary';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { a11yDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { StateVersionFileFragment_stateVersion$key } from './__generated__/StateVersionFileFragment_stateVersion.graphql';
import { StateVersionFileQuery } from './__generated__/StateVersionFileQuery.graphql';

interface Props {
    fragmentRef: StateVersionFileFragment_stateVersion$key
}

function StateVersionFileContainer(props: Props) {
    return (
        <ErrorBoundary fallbackRender={({ error }) => {
            if (!error?.codes?.includes('FORBIDDEN')) {
                throw error;
            }
            return (
                <Paper variant="outlined" sx={{ padding: 4, mt: 4, display: 'flex', justifyContent: 'center' }}>
                    <Typography color="textSecondary">You do not have permission to view state data for this workspace</Typography>
                </Paper>
            );
        }}>
            <StateVersionFile {...props} />
        </ErrorBoundary>
    );
}

function StateVersionFile(props: Props) {
    const { fragmentRef } = props;

    const data = useFragment<StateVersionFileFragment_stateVersion$key>(
        graphql`
      fragment StateVersionFileFragment_stateVersion on StateVersion
      {
        id
      }
    `, fragmentRef);

    const queryData = useLazyLoadQuery<StateVersionFileQuery>(graphql`
        query StateVersionFileQuery($id: String!) {
            node(id: $id) {
                ... on StateVersion {
                    data
                }
            }
        }
    `, { id: data.id }, { fetchPolicy: 'store-and-network' });

    const stateFileData = queryData.node?.data as string;

    return (
        <Box sx={{ fontSize: 14, overflowX: 'auto' }}>
            <SyntaxHighlighter language="json" style={a11yDark}>
                {JSON.stringify(JSON.parse(atob(stateFileData)), null, 2)}
            </SyntaxHighlighter>
        </Box>
    );
}

export default StateVersionFileContainer;
