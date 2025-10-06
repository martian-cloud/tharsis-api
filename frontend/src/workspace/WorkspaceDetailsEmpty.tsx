import { Paper, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { WorkspaceDetailsEmptyFragment_workspace$key } from './__generated__/WorkspaceDetailsEmptyFragment_workspace.graphql';

interface Props {
    fragmentRef: WorkspaceDetailsEmptyFragment_workspace$key
}

function WorkspaceDetailsEmpty(props: Props) {
    const { fragmentRef } = props;

    const data = useFragment<WorkspaceDetailsEmptyFragment_workspace$key>(
        graphql`
      fragment WorkspaceDetailsEmptyFragment_workspace on Workspace
      {
        id
        fullPath
      }
    `, fragmentRef);

    return (
        <React.Fragment>
            <Paper variant="outlined" sx={{ marginTop: 4, display: 'flex', justifyContent: 'center', marginBottom: 4 }}>
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center">
                    <Typography color="textSecondary" variant="h6" align="center" gutterBottom>
                        This workspace is empty
                    </Typography>
                    <Typography color="textSecondary" align="center">
                        You can get started by creating a run which will update the state of this workspace after the apply stage is complete
                    </Typography>
                </Box>
            </Paper>

            <Typography variant="h6" color="textSecondary" sx={{ marginBottom: 2 }}>Get started using the CLI</Typography>

            <Box mb={2}>
                <Typography variant="subtitle1">Login using your SSO credentials</Typography>
                <Paper>
                    <Box component="pre" padding={2} marginTop={0} marginBottom={0}>
                        tharsis sso login
                    </Box>
                </Paper>
            </Box>

            <Box mb={2}>
                <Typography variant="subtitle1">Apply a local configuration version</Typography>
                <Paper>
                    <Box component="pre" padding={2} marginTop={0} marginBottom={0} whiteSpace="pre-line">
                        tharsis apply --directory-path &lt;path-to-configuration-version&gt; {data.fullPath}
                    </Box>
                </Paper>
            </Box>

            <Box>
                <Typography variant="subtitle1">Apply a module from a module registry</Typography>
                <Paper>
                    <Box component="pre" padding={2} marginTop={0} marginBottom={0} whiteSpace="pre-line">
                        tharsis apply --module-source &lt;module-source-path&gt; {data.fullPath}
                    </Box>
                </Paper>
            </Box>
        </React.Fragment>
    );
}

export default WorkspaceDetailsEmpty;
