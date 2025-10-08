import { useState } from 'react';
import graphql from 'babel-plugin-relay/macro'
import { ConnectionHandler, useMutation, usePaginationFragment } from "react-relay/hooks";
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, Paper, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import ListSkeleton from '../skeletons/ListSkeleton';
import { LoadingButton } from '@mui/lab';
import { useSnackbar } from 'notistack';
import SyntaxHighlighter from 'react-syntax-highlighter';
import { a11yDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import TerraformModuleVersionAttestListItem from './TerraformModuleVersionAttestListItem';
import { TerraformModuleVersionAttestListFragment_attestations$key } from './__generated__/TerraformModuleVersionAttestListFragment_attestations.graphql';
import { TerraformModuleVersionAttestListDeleteMutation } from './__generated__/TerraformModuleVersionAttestListDeleteMutation.graphql';
import { TerraformModuleVersionAttestListPaginationQuery } from './__generated__/TerraformModuleVersionAttestListPaginationQuery.graphql';
import InfiniteScroll from 'react-infinite-scroll-component';

export const INITIAL_ITEM_COUNT = 50;

interface Props {
    fragmentRef: TerraformModuleVersionAttestListFragment_attestations$key
}

interface DataDialogProps {
    onCloseDataDialog: () => void
    encodedData: string | null
}

interface DeleteDialogProps {
    deleteInProgress: boolean;
    onCloseDeleteDialog: (confirm?: boolean) => void
    attestId: string | null
}

function GetConnections(id: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        id,
        'TerraformModuleVersionAttestList_attestations',
    );
    return [connectionId];
}

function DataDialog({ onCloseDataDialog, encodedData }: DataDialogProps) {

    return encodedData ?
        <Dialog
            maxWidth="md"
            open={!!encodedData}
        >
            <DialogTitle>Payload Data</DialogTitle>
            <DialogContent dividers>
                <Box sx={{ fontSize: 14, overflowX: 'auto' }}>
                    <SyntaxHighlighter language="json" style={a11yDark}>
                        {JSON.stringify(JSON.parse(atob(JSON.parse(atob(encodedData))['payload'])), null, 2)}
                    </SyntaxHighlighter>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button
                    size="small"
                    variant="outlined"
                    color="inherit"
                    onClick={onCloseDataDialog}>Close</Button>
            </DialogActions>
        </Dialog>
        :
    null
}

function DeleteConfirmationDialog({ deleteInProgress, onCloseDeleteDialog, attestId }: DeleteDialogProps) {

    return attestId ?
        <Dialog
            maxWidth="sm"
            keepMounted
            open={!!attestId}
        >
            <DialogTitle>Delete Attestation</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete attestation {attestId.substring(0, 8)}...?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onCloseDeleteDialog()}>
                    Cancel
                </Button>
                <LoadingButton color="error"
                    loading={deleteInProgress}
                    onClick={() => onCloseDeleteDialog(true)}
                >Delete</LoadingButton>
            </DialogActions>
        </Dialog>
        :
    null
}

function TerraformModuleVersionAttestList({ fragmentRef }: Props) {
    const theme = useTheme();
    const { enqueueSnackbar } = useSnackbar();
    const [attestToDelete, setAttestToDelete] = useState<string | null>(null);
    const [attestationDataToDisplay, setAttestationDataToDisplay] = useState<string | null>(null);

    const { data, loadNext, hasNext } = usePaginationFragment<TerraformModuleVersionAttestListPaginationQuery, TerraformModuleVersionAttestListFragment_attestations$key>(
        graphql`
            fragment TerraformModuleVersionAttestListFragment_attestations on TerraformModuleVersion
                @refetchable(queryName: "TerraformModuleVersionAttestListPaginationQuery") {
                    id
                    attestations(
                        first: $first
                        after: $after
                    ) @connection(key: "TerraformModuleVersionAttestList_attestations"){
                        edges {
                            node {
                                id
                                data
                                ...TerraformModuleVersionAttestListItemFragment_module
                            }
                        }
                    }
                }
            `, fragmentRef);

    const [commitDeleteAttestation, commitInFlight] = useMutation<TerraformModuleVersionAttestListDeleteMutation>(graphql`
        mutation TerraformModuleVersionAttestListDeleteMutation($input: DeleteTerraformModuleAttestationInput!, $connections: [ID!]!) {
            deleteTerraformModuleAttestation(input: $input) {
                moduleAttestation {
                    id @deleteEdge(connections: $connections)
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const onCloseDeleteAttestConfirmation = (confirm?: boolean) => {
        if (confirm && attestToDelete) {
            commitDeleteAttestation({
                variables: {
                    input: {
                        id: attestToDelete
                    },
                    connections: GetConnections(data.id)
                },
                onCompleted: data => {
                    if (data.deleteTerraformModuleAttestation.problems.length) {
                        enqueueSnackbar(data.deleteTerraformModuleAttestation.problems.map(problem => problem.message).join('; '),  { variant: 'warning' });
                    }
                    setAttestToDelete(null)
                },
                onError: error => {
                    setAttestToDelete(null)
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setAttestToDelete(null)
        }
    };

    return (data.attestations.edges && data.attestations.edges.length > 0) ?
        <Box>
            <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                    <Typography variant="subtitle1">
                        {data.attestations?.edges?.length} attestation{data.attestations.edges.length === 1 ? '' : 's'}
                    </Typography>
                </Box>
            </Paper>
            <InfiniteScroll
                dataLength={data.attestations.edges.length ?? 0}
                next={() => loadNext(20)}
                hasMore={hasNext}
                loader={<ListSkeleton rowCount={3} />}
            >
                <TableContainer>
                    <Table sx={{ tableLayout: 'fixed' }}>
                        <colgroup>
                            <Box component="col" />
                            <Box component="col" />
                            <Box component="col" />
                            <Box component="col" />
                            <Box component="col" sx={{ width: '175px' }} />
                        </colgroup>
                        <TableHead>
                            <TableRow>
                                <TableCell>
                                    <Typography color="textSecondary">ID</Typography>
                                </TableCell>
                                <TableCell>
                                    <Typography color="textSecondary">Description</Typography>
                                </TableCell>
                                <TableCell>
                                    <Typography color="textSecondary">Predicate Type</Typography>
                                </TableCell>
                                <TableCell>
                                    <Typography color="textSecondary">Created</Typography>
                                </TableCell>
                                <TableCell>
                                    <Typography color="textSecondary">Actions</Typography>
                                </TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {data.attestations.edges?.map((edge: any) => <TerraformModuleVersionAttestListItem
                                key={edge.node.id}
                                fragmentRef={edge.node}
                                onOpenDataDialog={() => setAttestationDataToDisplay(edge.node.data)}
                                onOpenDeleteDialog={() => setAttestToDelete(edge.node.id)}
                            />)}
                        </TableBody>
                    </Table>
                </TableContainer>
            </InfiniteScroll>
            <DataDialog
                encodedData={attestationDataToDisplay}
                onCloseDataDialog={() => setAttestationDataToDisplay(null)} />
            <DeleteConfirmationDialog
                deleteInProgress={commitInFlight}
                onCloseDeleteDialog={onCloseDeleteAttestConfirmation}
                attestId={attestToDelete}
            />
        </Box>
        :
        <Box padding={2} display="flex" justifyContent="center" alignItems="center">
			<Typography color="textSecondary">No attestations for this version</Typography>
		</Box>
}

export default TerraformModuleVersionAttestList
