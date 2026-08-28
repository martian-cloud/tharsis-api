import { CircularProgress, Paper, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import ToggleButton from '@mui/material/ToggleButton';
import ToggleButtonGroup from '@mui/material/ToggleButtonGroup';
import graphql from 'babel-plugin-relay/macro';
import { Suspense, useState } from 'react';
import { ErrorBoundary } from 'react-error-boundary';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { a11yDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import { decodeBase64Utf8 } from '../../common/base64';
import { StateVersionFileFragment_stateVersion$key } from './__generated__/StateVersionFileFragment_stateVersion.graphql';
import { StateVersionFileJSONQuery } from './__generated__/StateVersionFileJSONQuery.graphql';
import { StateVersionFileQuery } from './__generated__/StateVersionFileQuery.graphql';

interface Props {
    fragmentRef: StateVersionFileFragment_stateVersion$key
}

// StateFormat selects which of the two stored representations of the state is shown. 'raw' is
// Terraform's own state file; 'json' is the documented "terraform show -json" representation, which
// is stored as a separate artifact and may be absent (state pushed by a client that does not produce
// one, or written by a runner predating it).
type StateFormat = 'raw' | 'json';

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

// StateJSON is a separate component so its query is only issued once the JSON format is actually
// selected: rendering it on mount would fetch a second, typically larger, copy of the state that most
// visitors never look at. Relay keeps the result once fetched, so toggling back and forth is free.
function StateJSON({ id }: { id: string }) {
    const queryData = useLazyLoadQuery<StateVersionFileJSONQuery>(graphql`
        query StateVersionFileJSONQuery($id: String!) {
            node(id: $id) {
                ... on StateVersion {
                    jsonData
                }
            }
        }
    `, { id }, { fetchPolicy: 'store-or-network' });

    const jsonData = queryData.node?.jsonData;

    if (!jsonData) {
        return (
            <Paper variant="outlined" sx={{ padding: 4, display: 'flex', justifyContent: 'center' }}>
                <Typography color="textSecondary">
                    No JSON representation was stored for this state version
                </Typography>
            </Paper>
        );
    }

    return <StateContent json={jsonData} />;
}

// StateContent pretty-prints a state document. It takes JSON text, so decoding is the caller's
// concern: the raw state arrives base64-encoded because the API treats it as an opaque blob, while the
// representation arrives as text.
function StateContent({ json }: { json: string }) {
    return (
        <Box sx={{ fontSize: 14, overflowX: 'auto' }}>
            <SyntaxHighlighter language="json" style={a11yDark} customStyle={{marginTop: 0}}>
                {JSON.stringify(JSON.parse(json), null, 2)}
            </SyntaxHighlighter>
        </Box>
    );
}

function StateVersionFile(props: Props) {
    const { fragmentRef } = props;
    const [format, setFormat] = useState<StateFormat>('raw');

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

    const onFormatChange = (_event: React.MouseEvent<HTMLElement>, value: StateFormat | null) => {
        // ToggleButtonGroup reports null when the active button is clicked again; keep the current
        // format rather than leaving no format selected.
        if (value) {
            setFormat(value);
        }
    };

    return (
        <Box sx={{ position: 'relative', mt: 2 }}>
            {/*
              * Overlaid on the panel's top right corner rather than sitting above it. The control is
              * anchored to this container, not to the scrolling content, so it stays put while long
              * lines scroll horizontally underneath. The paper background keeps it legible where it
              * covers highlighted code.
              */}
            <ToggleButtonGroup
                size="small"
                color="primary"
                value={format}
                exclusive
                onChange={onFormatChange}
                sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                    zIndex: 1,
                    backgroundColor: 'background.paper',
                }}
            >
                <ToggleButton value="raw" size="small">Raw</ToggleButton>
                <ToggleButton value="json" size="small">Enhanced</ToggleButton>
            </ToggleButtonGroup>
            {format === 'raw' && <StateContent json={decodeBase64Utf8(stateFileData)} />}
            {format === 'json' && <Suspense fallback={
                <Box sx={{ minHeight: 120, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    <CircularProgress />
                </Box>
            }>
                <StateJSON id={data.id} />
            </Suspense>}
        </Box>
    );
}

export default StateVersionFileContainer;
