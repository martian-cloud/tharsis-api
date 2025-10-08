import LoadingButton from '@mui/lab/LoadingButton';
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, List, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import InfiniteScroll from 'react-infinite-scroll-component';
import { ConnectionHandler, useFragment, useLazyLoadQuery, useMutation, usePaginationFragment } from "react-relay/hooks";
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ListSkeleton from '../../skeletons/ListSkeleton';
import GPGKeyListItem from './GPGKeyListItem';
import { GPGKeyListDeleteMutation } from './__generated__/GPGKeyListDeleteMutation.graphql';
import { GPGKeyListFragment_group$key } from './__generated__/GPGKeyListFragment_group.graphql';
import { GPGKeyListFragment_keys$key } from './__generated__/GPGKeyListFragment_keys.graphql';
import { GPGKeyListPaginationQuery } from './__generated__/GPGKeyListPaginationQuery.graphql';
import { GPGKeyListQuery } from './__generated__/GPGKeyListQuery.graphql';

const DESCRIPTION = 'GPG Keys are required when publishing Terraform providers to the provider registry';
const INITIAL_ITEM_COUNT = 100;

const query = graphql`
    query GPGKeyListQuery($first: Int, $last: Int, $after: String, $before: String, $groupId: String!) {
        node(id: $groupId) {
            ...on Group {
                ...GPGKeyListFragment_keys
            }
        }
    }
`;

interface ConfirmationDialogProps {
    gpgKeyId: string
    deleteInProgress: boolean;
    onClose: (confirm?: boolean) => void
}

function DeleteConfirmationDialog(props: ConfirmationDialogProps) {
    const { gpgKeyId, deleteInProgress, onClose, ...other } = props;
    return (
        <Dialog
            maxWidth="xs"
            keepMounted={false}
            open={true}
            {...other}
        >
            <DialogTitle>Delete GPG Key</DialogTitle>
            <DialogContent dividers>
                Are you sure you want to delete GPG key <strong>{gpgKeyId}</strong>?
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton color="error" loading={deleteInProgress} onClick={() => onClose(true)}>
                    Delete
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

interface Props {
    fragmentRef: GPGKeyListFragment_group$key
}

export function GetConnections(groupId: string): [string] {
    const connectionId = ConnectionHandler.getConnectionID(
        groupId,
        'GPGKeyList_gpgKeys',
        { includeInherited: false }
    );
    return [connectionId];
}

function GPGKeyList(props: Props) {
    const theme = useTheme();
    const { enqueueSnackbar } = useSnackbar();
    const navigate = useNavigate();

    const [gpgKeyToDelete, setGPGKeyToDelete] = useState<any>(null);

    const group = useFragment<GPGKeyListFragment_group$key>(
        graphql`
        fragment GPGKeyListFragment_group on Group
        {
          id
          fullPath
        }
    `, props.fragmentRef);

    const queryData = useLazyLoadQuery<GPGKeyListQuery>(query, { first: INITIAL_ITEM_COUNT, groupId: group.id }, { fetchPolicy: 'store-and-network' });

    const { data, loadNext, hasNext } = usePaginationFragment<GPGKeyListPaginationQuery, GPGKeyListFragment_keys$key>(
        graphql`
      fragment GPGKeyListFragment_keys on Group
      @refetchable(queryName: "GPGKeyListPaginationQuery") {
            gpgKeys(
                after: $after
                before: $before
                first: $first
                last: $last
                includeInherited: true
                sort: GROUP_LEVEL_DESC
            ) @connection(key: "GPGKeyList_gpgKeys") {
                totalCount
                edges {
                    node {
                        id
                        gpgKeyId
                        groupPath
                        ...GPGKeyListItemFragment_key
                    }
                }
            }
      }
    `, queryData.node);

    const [commit, commitInFlight] = useMutation<GPGKeyListDeleteMutation>(graphql`
        mutation GPGKeyListDeleteMutation($input: DeleteGPGKeyInput!, $connections: [ID!]!) {
            deleteGPGKey(input: $input) {
                gpgKey {
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

    const onDeleteConfirmationDialogClosed = (confirm?: boolean) => {
        if (confirm) {
            commit({
                variables: {
                    input: {
                        id: gpgKeyToDelete.id
                    },
                    connections: GetConnections(group.id),
                },
                onCompleted: data => {
                    setGPGKeyToDelete(null);

                    if (data.deleteGPGKey.problems.length) {
                        enqueueSnackbar(data.deleteGPGKey.problems.map((problem: any) => problem.message).join('; '), { variant: 'warning' });
                    } else {
                        navigate(`..`);
                    }
                },
                onError: error => {
                    setGPGKeyToDelete(null);
                    enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            });
        } else {
            setGPGKeyToDelete(null);
        }
    };


    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "keys", path: 'keys' }
                ]}
            />
            {data?.gpgKeys.edges?.length !== 0 && <Box>
                <Box marginBottom={2}>
                    <Box sx={{
                        display: 'flex',
                        flexDirection: 'row',
                        justifyContent: 'space-between',
                        [theme.breakpoints.down('md')]: {
                            flexDirection: 'column',
                            alignItems: 'flex-start',
                            '& > *': { marginBottom: 2 },
                        }
                    }}>
                        <Box>
                            <Typography variant="h5" gutterBottom>GPG Keys</Typography>
                            <Typography variant="body2">
                                {DESCRIPTION}
                            </Typography>
                        </Box>
                        <Box>
                            <Button sx={{ minWidth: 150 }} component={RouterLink} variant="outlined" to="new">
                                New GPG Key
                            </Button>
                        </Box>
                    </Box>
                </Box>
                <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
                    <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
                        <Typography variant="subtitle1">
                            {data?.gpgKeys.totalCount} key{data?.gpgKeys.totalCount === 1 ? '' : 's'}
                        </Typography>
                    </Box>
                </Paper>
                <InfiniteScroll
                    dataLength={data?.gpgKeys.edges?.length ?? 0}
                    next={() => loadNext(20)}
                    hasMore={hasNext}
                    loader={<ListSkeleton rowCount={3} />}
                >
                    <List disablePadding>
                        {data?.gpgKeys.edges?.map((edge: any) => <GPGKeyListItem
                            key={edge.node.id}
                            fragmentRef={edge.node}
                            onDelete={() => setGPGKeyToDelete(edge.node)}
                            inherited={edge.node.groupPath !== group.fullPath}
                        />)}
                    </List>
                </InfiniteScroll>
            </Box>}
            {data?.gpgKeys.edges?.length === 0 && <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                    <Typography variant="h6">This group does not have any GPG Keys</Typography>
                    <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                        {DESCRIPTION}
                    </Typography>
                    <Button component={RouterLink} variant="outlined" to="new">New GPG Key</Button>
                </Box>
            </Box>}
            {gpgKeyToDelete && <DeleteConfirmationDialog gpgKeyId={gpgKeyToDelete.gpgKeyId} deleteInProgress={commitInFlight} onClose={onDeleteConfirmationDialogClosed} />}
        </Box>
    );
}

export default GPGKeyList;
