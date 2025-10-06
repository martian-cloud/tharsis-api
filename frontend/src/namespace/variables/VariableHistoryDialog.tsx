import LockIcon from '@mui/icons-material/LockOutlined';
import {
    Box,
    Button,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Link,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    useMediaQuery,
    useTheme
} from '@mui/material';
import graphql from "babel-plugin-relay/macro";
import React, { Suspense } from 'react';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import DataTableCell from '../../common/DataTableCell';
import Timestamp from '../../common/Timestamp';
import { VariableHistoryDialogFragment_variable$key } from './__generated__/VariableHistoryDialogFragment_variable.graphql';
import { VariableHistoryDialogQuery } from './__generated__/VariableHistoryDialogQuery.graphql';
import SensitiveVariableValue from './SensitiveVariableValue';

const INITIAL_ITEM_COUNT = 5;

function VariableHistory({ variableId, sensitive }: { variableId: string, sensitive: boolean }) {
    const variable = useLazyLoadQuery<VariableHistoryDialogQuery>(graphql`
    query VariableHistoryDialogQuery($id: String!, $first: Int!, $after: String, $includeValues: Boolean!) {
        node(id: $id) {
            ... on NamespaceVariable {
                sensitive
                ...VariableHistoryDialogFragment_variable
            }
        }
    }`, { id: variableId, first: INITIAL_ITEM_COUNT, includeValues: !sensitive }, { fetchPolicy: 'network-only' });

    const { data, loadNext, hasNext } = usePaginationFragment<VariableHistoryDialogQuery, VariableHistoryDialogFragment_variable$key>(
        graphql`
        fragment VariableHistoryDialogFragment_variable on NamespaceVariable
        @refetchable(queryName: "VariableHistoryDialogPaginationQuery") {
            versions(
                first: $first
                after: $after
                sort: CREATED_AT_DESC
                ) @connection(key: "VariableHistoryDialog_versions") {
                    totalCount
                    edges {
                        node {
                            metadata {
                                createdAt
                            }
                            id
                            key
                            value @include(if: $includeValues)
                            hcl
                        }
                    }
                }
            }
        `, variable.node);

    return (
        <Box>
            <TableContainer>
                <Table>
                    <TableHead>
                        <TableRow>
                            <TableCell>Created</TableCell>
                            <TableCell>key</TableCell>
                            <TableCell>Value</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {data?.versions.edges?.map((edge) => <TableRow key={edge?.node?.id}>
                            <TableCell>
                                <Timestamp timestamp={edge?.node?.metadata.createdAt} />
                            </TableCell>
                            <DataTableCell sx={{ wordBreak: 'break-all' }} >
                                {edge?.node?.key}
                            </DataTableCell>
                            <DataTableCell sx={{ wordBreak: 'break-all' }} >
                                {edge?.node?.value === null && !sensitive && <LockIcon color="disabled" />}
                                {edge?.node?.value !== null && !sensitive && <React.Fragment>
                                    {edge?.node?.value}
                                </React.Fragment>}
                                {sensitive && <SensitiveVariableValue variableVersionId={edge?.node?.id as string} />}
                            </DataTableCell>
                        </TableRow>)}
                    </TableBody>
                </Table>
            </TableContainer>
            {hasNext && <Link
                mt={2}
                component="div"
                variant="body2"
                color="textSecondary"
                sx={{ cursor: 'pointer' }}
                underline="hover"
                onClick={() => loadNext(INITIAL_ITEM_COUNT)}
            >
                Show more
            </Link>}
        </Box>
    );
}

interface Props {
    variableId: string
    sensitive: boolean
    onClose: (keepOpen: boolean) => void
}

function VariableHistoryDialog({ variableId, sensitive, onClose }: Props) {
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

    return (
        <Dialog
            fullWidth
            maxWidth="md"
            fullScreen={fullScreen}
            open
        >
            <DialogTitle>Variable History</DialogTitle>
            <DialogContent dividers sx={{ flex: 1, padding: 2, minHeight: 400, display: 'flex', flexDirection: 'column' }}>
                <Suspense fallback={<Box
                    sx={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        width: '100%',
                        minHeight: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}>
                    <CircularProgress />
                </Box>}>
                    <VariableHistory variableId={variableId} sensitive={sensitive} />
                </Suspense>
            </DialogContent>
            <DialogActions sx={{ pl: 3, pr: 3, justifyContent: 'flex-end' }}>
                <Stack direction="row" spacing={2}>
                    <Button onClick={() => onClose(false)} color="inherit">
                        Close
                    </Button>
                </Stack>
            </DialogActions>
        </Dialog>
    );
}

export default VariableHistoryDialog;
