import { Alert, Box, Divider, TextField, Typography } from "@mui/material";
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { VCSWorkspaceLinkSourceFragment_workspace$key } from "./__generated__/VCSWorkspaceLinkSourceFragment_workspace.graphql";

export type VCSRunDataOptions = {
    referenceName: string | null;
};

export const DefaultVCSRunDataOptions: VCSRunDataOptions = {
    referenceName: null
};

interface Props {
    data: VCSRunDataOptions
    onChange: (value: VCSRunDataOptions) => void
    fragmentRef: VCSWorkspaceLinkSourceFragment_workspace$key
}

function VCSWorkspaceLinkSource({ data, onChange, fragmentRef }: Props) {

    const workspace = useFragment<VCSWorkspaceLinkSourceFragment_workspace$key>(
        graphql`
        fragment VCSWorkspaceLinkSourceFragment_workspace on Workspace
        {
            workspaceVcsProviderLink {
                branch
            }
        }
    `, fragmentRef);

    return (
        workspace.workspaceVcsProviderLink ?
            <Box mb={4}>
                <Typography variant="subtitle1" gutterBottom>Select Branch</Typography>
                <Divider light />
                <Box mt={2}>
                    <TextField
                        autoComplete="off"
                        size="small"
                        fullWidth
                        label="Branch"
                        defaultValue={workspace.workspaceVcsProviderLink.branch}
                        onChange={(event: any) => onChange({ ...data, referenceName: event.target.value })}
                    />
                    <Typography color="textSecondary" variant="caption" mt={1}>OPTIONAL: Enter the branch name in the repository that will trigger this run. If no branch name is entered, the run will be triggered in the VCS workspace link's default branch.</Typography>
                </Box>
            </Box>
            :
            <Alert sx={{ mb: 4 }} severity="warning">
                This option is not available because this workspace is not linked to a VCS Provider.
            </Alert>
    );
}

export default VCSWorkspaceLinkSource
