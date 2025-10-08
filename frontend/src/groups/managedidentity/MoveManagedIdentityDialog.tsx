import { LoadingButton } from "@mui/lab";
import {
    Alert,
    AlertTitle,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
} from "@mui/material";
import graphql from "babel-plugin-relay/macro";
import { useSnackbar } from "notistack";
import { useCallback, useState } from "react";
import { useFragment, useMutation } from "react-relay";
import { useNavigate } from "react-router-dom";
import { MutationError } from "../../common/error";
import { GetConnections } from "./ManagedIdentityList";
import GroupAutocomplete, { GroupOption } from "../GroupAutocomplete";
import { MoveManagedIdentityDialogFragment_managedIdentity$key } from "./__generated__/MoveManagedIdentityDialogFragment_managedIdentity.graphql";
import { MoveManagedIdentityDialogMutation } from "./__generated__/MoveManagedIdentityDialogMutation.graphql";

interface Props {
    onClose: () => void;
    fragmentRef: MoveManagedIdentityDialogFragment_managedIdentity$key;
    groupId: string;
}

function MoveManagedIdentityDialog({ onClose, fragmentRef, groupId }: Props) {
    const navigate = useNavigate();
    const { enqueueSnackbar } = useSnackbar();
    const [newParentPath, setNewParentPath] = useState<string>("");
    const [error, setError] = useState<MutationError>();

    const managedIdentity =
        useFragment<MoveManagedIdentityDialogFragment_managedIdentity$key>(
            graphql`
                fragment MoveManagedIdentityDialogFragment_managedIdentity on ManagedIdentity {
                    id
                    name
                    groupPath
                }
            `,
            fragmentRef
        );

    const [commit, isInFlight] =
        useMutation<MoveManagedIdentityDialogMutation>(graphql`
            mutation MoveManagedIdentityDialogMutation(
                $input: MoveManagedIdentityInput!
                $connections: [ID!]!
            ) {
                moveManagedIdentity(input: $input) {
                    managedIdentity {
                        id @deleteEdge(connections: $connections)
                        groupPath
                    }
                    problems {
                        message
                        field
                        type
                    }
                }
            }
        `);

    const filterGroups = useCallback(
        (options: readonly GroupOption[]) => {
            return options.filter(
                (opt: GroupOption) => opt.fullPath !== managedIdentity.groupPath
            );
        },
        [managedIdentity]
    );

    const onMove = () => {
        commit({
            variables: {
                input: {
                    managedIdentityId: managedIdentity.id,
                    newParentPath: newParentPath,
                },
                connections: GetConnections(groupId),
            },
            onCompleted: (data) => {
                if (data.moveManagedIdentity.problems.length) {
                    setError({
                        severity: "warning",
                        message: data.moveManagedIdentity.problems
                            .map((problem) => problem.message)
                            .join("; "),
                    });
                } else if (!data.moveManagedIdentity.managedIdentity) {
                    setError({
                        severity: "error",
                        message: "Unexpected error occurred",
                    });
                } else {
                    onClose();
                    navigate(
                        `/groups/${data.moveManagedIdentity.managedIdentity.groupPath}/-/managed_identities/${data.moveManagedIdentity.managedIdentity.id}`
                    );
                    enqueueSnackbar(
                        `${managedIdentity.name} has been moved to group ${data.moveManagedIdentity.managedIdentity.groupPath}`,
                        { variant: "success" }
                    );
                }
            },
            onError: (error) => {
                setError({
                    severity: "error",
                    message: `Unexpected error occurred: ${error.message}`,
                });
            },
        });
    };

    const onGroupChange = (group: any) => {
        setNewParentPath(group?.fullPath);
    };

    return (
        <Dialog keepMounted maxWidth="sm" open>
            <DialogTitle>Move Managed Identity</DialogTitle>
            <DialogContent dividers>
                {error && (
                    <Alert sx={{ mb: 2 }} severity={error.severity}>
                        {error.message}
                    </Alert>
                )}
                <Alert sx={{ mb: 2 }} severity="warning">
                    <AlertTitle>Warning</AlertTitle>
                    Managed identity <strong>cannot</strong> be moved if target group contains its alias(es) or if it's assigned to workspace(s) outside of the target group.
                </Alert>
                <GroupAutocomplete
                    placeholder="Select a group"
                    sx={{ mb: 2 }}
                    onSelected={onGroupChange}
                    filterGroups={filterGroups}
                />
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={onClose}>
                    Cancel
                </Button>
                <LoadingButton
                    disabled={!newParentPath}
                    color="primary"
                    variant="outlined"
                    loading={isInFlight}
                    onClick={onMove}
                >
                    Move
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default MoveManagedIdentityDialog;
