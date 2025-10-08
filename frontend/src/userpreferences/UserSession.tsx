import { Box, Typography, Chip } from '@mui/material';
import { LoadingButton } from '@mui/lab';
import { useFragment, useMutation } from 'react-relay';
import { useSnackbar } from 'notistack';
import graphql from 'babel-plugin-relay/macro';
import Timestamp from '../common/Timestamp';
import { UserSessionFragment_session$key } from './__generated__/UserSessionFragment_session.graphql';
import { UserSessionRevokeMutation } from './__generated__/UserSessionRevokeMutation.graphql';

interface Props {
    fragmentRef: UserSessionFragment_session$key;
}

function UserSession({ fragmentRef }: Props) {
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<UserSessionFragment_session$key>(
        graphql`
            fragment UserSessionFragment_session on UserSession {
                id
                userAgent
                expiration
                expired
                metadata {
                    createdAt
                }
            }
        `,
        fragmentRef
    );

    const [commit, isInFlight] = useMutation<UserSessionRevokeMutation>(graphql`
        mutation UserSessionRevokeMutation($input: RevokeUserSessionInput!) {
            revokeUserSession(input: $input) {
                problems {
                    message
                    field
                }
            }
        }
    `);

    const handleRevoke = () => {
        commit({
            variables: {
                input: {
                    userSessionId: data.id
                }
            },
            updater: (store) => {
                // Remove the session from the cache
                store.delete(data.id);
            },
            onCompleted: (response) => {
                if (response.revokeUserSession.problems.length) {
                    enqueueSnackbar(response.revokeUserSession.problems.map(p => p.message).join('; '), { variant: 'warning' });
                } else {
                    enqueueSnackbar('Session revoked successfully', { variant: 'success' });
                }
            },
            onError: () => {
                enqueueSnackbar('Failed to revoke session', { variant: 'error' });
            }
        });
    };

    return (
        <Box sx={{ p: 2, border: 1, borderColor: 'divider', borderRadius: 1, mb: 2 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
                <Box sx={{ flex: 1 }}>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', mb: 1 }}>
                        {data.userAgent}
                    </Typography>
                    <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mb: 1 }}>
                        <Typography variant="caption" color="text.secondary">
                            Created: <Timestamp timestamp={data.metadata.createdAt} />
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                            Expires: <Timestamp timestamp={data.expiration} />
                        </Typography>
                    </Box>
                    <Chip
                        label={data.expired ? 'Expired' : 'Active'}
                        color={data.expired ? 'default' : 'success'}
                        size="small"
                    />
                </Box>
                {!data.expired && (
                    <LoadingButton
                        variant="outlined"
                        color="error"
                        size="small"
                        onClick={handleRevoke}
                        loading={isInFlight}
                    >
                        Revoke
                    </LoadingButton>
                )}
            </Box>
        </Box>
    );
}

export default UserSession;
