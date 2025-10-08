import LoadingButton from '@mui/lab/LoadingButton';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ManagedIdentityForm, { FormData } from './ManagedIdentityForm';
import { EditManagedIdentityFragment_group$key } from './__generated__/EditManagedIdentityFragment_group.graphql';
import { EditManagedIdentityMutation } from './__generated__/EditManagedIdentityMutation.graphql';
import { EditManagedIdentityQuery } from './__generated__/EditManagedIdentityQuery.graphql';

interface Props {
    fragmentRef: EditManagedIdentityFragment_group$key
}

function parsePayloadData(data: string): any {
    return JSON.parse(atob(data));
}

function EditManagedIdentity(props: Props) {
    const { id } = useParams();
    const navigate = useNavigate();

    const managedIdentityId = id as string;

    const group = useFragment<EditManagedIdentityFragment_group$key>(
        graphql`
        fragment EditManagedIdentityFragment_group on Group
        {
          id
          fullPath
        }
      `, props.fragmentRef
    );

    const queryData = useLazyLoadQuery<EditManagedIdentityQuery>(graphql`
        query EditManagedIdentityQuery($id: String!) {
            managedIdentity(id: $id) {
                id
                type
                name
                description
                data
            }
        }
    `, { id: managedIdentityId })

    const [commit, isInFlight] = useMutation<EditManagedIdentityMutation>(graphql`
        mutation EditManagedIdentityMutation($input: UpdateManagedIdentityInput!) {
            updateManagedIdentity(input: $input) {
                managedIdentity {
                    id
                    data
                    ...ManagedIdentityListItemFragment_managedIdentity
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
    const [formData, setFormData] = useState<FormData | null>(queryData.managedIdentity ? {
        type: queryData.managedIdentity.type,
        name: queryData.managedIdentity.name,
        description: queryData.managedIdentity.description,
        payload: parsePayloadData(queryData.managedIdentity.data),
        rules: []
    } : null);

    const onUpdate = () => {
        if (formData) {
            commit({
                variables: {
                    input: {
                        id: managedIdentityId,
                        description: formData.description,
                        data: btoa(JSON.stringify(formData.payload)),
                    }
                },
                onCompleted: data => {
                    if (data.updateManagedIdentity.problems.length) {
                        setError({
                            severity: 'warning',
                            message: data.updateManagedIdentity.problems.map(problem => problem.message).join('; ')
                        });
                    } else if (!data.updateManagedIdentity.managedIdentity) {
                        setError({
                            severity: 'error',
                            message: "Unexpected error occurred"
                        });
                    } else {
                        navigate(`../${data.updateManagedIdentity.managedIdentity.id}`);
                    }
                },
                onError: error => {
                    setError({
                        severity: 'error',
                        message: `Unexpected error occurred: ${error.message}`
                    });
                }
            });
        }
    };

    return formData ? (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "managed identities", path: 'managed_identities' },
                    { title: formData.name, path: managedIdentityId },
                    { title: "edit", path: 'edit' },
                ]}
            />
            <Typography variant="h5">Edit Managed Identity</Typography>
            <ManagedIdentityForm
                groupPath={group.fullPath}
                editMode
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider light sx={{ marginTop: 4 }} />
            <Box marginTop={2}>
                <LoadingButton
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onUpdate}>
                    Update Managed Identity
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    ) : <Box>Not found</Box>;
}

export default EditManagedIdentity;
