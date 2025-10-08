import LoadingButton from '@mui/lab/LoadingButton';
import { Alert, TextField } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import { GetConnections } from './GPGKeyList';
import { NewGPGKeyFragment_group$key } from './__generated__/NewGPGKeyFragment_group.graphql';
import { NewGPGKeyMutation } from './__generated__/NewGPGKeyMutation.graphql';

interface Props {
    fragmentRef: NewGPGKeyFragment_group$key
}

function NewGPGKey(props: Props) {
    const navigate = useNavigate();

    const group = useFragment<NewGPGKeyFragment_group$key>(
        graphql`
        fragment NewGPGKeyFragment_group on Group
        {
          id
          fullPath
        }
      `,
        props.fragmentRef
    );

    const [commit, isInFlight] = useMutation<NewGPGKeyMutation>(graphql`
        mutation NewGPGKeyMutation($input: CreateGPGKeyInput!, $connections: [ID!]!) {
            createGPGKey(input: $input) {
                # Use @prependNode to add the node to the connection
                gpgKey  @prependNode(connections: $connections, edgeTypeName: "GPGKeyEdge")  {
                    id
                    gpgKeyId
                    fingerprint
                    createdBy
                    metadata {
                        createdAt
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [error, setError] = useState<MutationError>()
    const [pubKey, setPubKey] = useState('');

    const onSave = () => {
        commit({
            variables: {
                input: {
                    groupPath: group.fullPath,
                    asciiArmor: pubKey
                },
                connections: GetConnections(group.id)
            },
            onCompleted: data => {
                if (data.createGPGKey.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createGPGKey.problems.map((problem: any) => problem.message).join('; ')
                    });
                } else if (!data.createGPGKey.gpgKey) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`..`);
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        });
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "keys", path: 'keys' },
                    { title: "new", path: 'new' },
                ]}
            />
            <Typography variant="h5">New GPG Key</Typography>
            <Box marginBottom={2} marginTop={2}>
                {error && <Alert sx={{ marginTop: 2, marginBottom: 2 }} severity={error.severity}>
                    {error.message}
                </Alert>}
                <Typography gutterBottom>Public Key</Typography>
                <TextField
                    margin='none'
                    fullWidth
                    placeholder={`Paste your public key here starting with -----BEGIN PGP PUBLIC KEY BLOCK-----`}
                    value={pubKey}
                    multiline
                    rows={15}
                    onChange={event => setPubKey(event.target.value)}
                />
            </Box>
            <Divider light />
            <Box marginTop={2}>
                <LoadingButton
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onSave}>
                    Create Key
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewGPGKey;
