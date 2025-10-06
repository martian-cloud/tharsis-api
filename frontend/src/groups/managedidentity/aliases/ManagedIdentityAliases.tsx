import { useState } from "react";
import { Box, Button, Typography } from "@mui/material";
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay/hooks";
import { ManagedIdentityAliasesFragment_managedIdentity$key } from "./__generated__/ManagedIdentityAliasesFragment_managedIdentity.graphql";
import NewManagedIdentityAliasDialog from "./NewManagedIdentityAliasDialog";
import ManagedIdentityAliasesList from "./ManagedIdentityAliasesList";

interface Props {
    fragmentRef: ManagedIdentityAliasesFragment_managedIdentity$key
}

function ManagedIdentityAliases({ fragmentRef }: Props) {
    const [showCreateNewAliasDialog, setShowCreateNewAliasDialog] = useState<boolean>(false)

    const data = useFragment<ManagedIdentityAliasesFragment_managedIdentity$key>(
        graphql`
            fragment ManagedIdentityAliasesFragment_managedIdentity on ManagedIdentity
            {
                ...ManagedIdentityAliasesListFragment_managedIdentity
                ...NewManagedIdentityAliasDialogFragment_managedIdentity
            }
        `, fragmentRef
    );

    return (
        <Box>
            <Typography sx={{ marginBottom: 2 }} color="textSecondary">A managed identity alias points to an existing managed identity, which allows it to be used in a group outside the managed identity's group hierarchy.
            </Typography>
            <ManagedIdentityAliasesList fragmentRef={data} />
            <Button
                sx={{ mt: 3 }}
                color="secondary"
                size="small"
                variant="outlined"
                onClick={() => setShowCreateNewAliasDialog(true)}>Create Alias
            </Button>
            {showCreateNewAliasDialog && <NewManagedIdentityAliasDialog fragmentRef={data}
                onClose={() => setShowCreateNewAliasDialog(false)} />}
        </Box>
    );
}

export default ManagedIdentityAliases
