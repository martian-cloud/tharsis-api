import LoadingButton from '@mui/lab/LoadingButton';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useFragment, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import ManagedIdentityForm, { FormData } from './ManagedIdentityForm';
import { GetConnections } from './ManagedIdentityList';
import { NewManagedIdentityFragment_group$key } from './__generated__/NewManagedIdentityFragment_group.graphql';
import { NewManagedIdentityMutation } from './__generated__/NewManagedIdentityMutation.graphql';

interface Props {
    fragmentRef: NewManagedIdentityFragment_group$key
}

function NewManagedIdentity(props: Props) {
    const navigate = useNavigate();
    const [error, setError] = React.useState<MutationError>()
    const [formData, setFormData] = useState<FormData>({
        type: '',
        name: '',
        description: '',
        payload: {},
        rules: []
    });

    const group = useFragment<NewManagedIdentityFragment_group$key>(
        graphql`
        fragment NewManagedIdentityFragment_group on Group
        {
          id
          fullPath
        }
      `, props.fragmentRef);

    const [commit, isInFlight] = useMutation<NewManagedIdentityMutation>(graphql`
        mutation NewManagedIdentityMutation($input: CreateManagedIdentityInput!, $connections: [ID!]!) {
            createManagedIdentity(input: $input) {
                # Use @prependNode to add the node to the connection
                managedIdentity @prependNode(connections: $connections, edgeTypeName: "ManagedIdentityEdge") {
                    id
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

    const onCreate = () => {
        commit({
            variables: {
                input: {
                    type: formData.type,
                    name: formData.name,
                    description: formData.description,
                    data: btoa(JSON.stringify(formData.payload)),
                    accessRules: formData.rules.map(rule => ({
                        type: rule.type,
                        runStage: rule.runStage,
                        allowedServiceAccounts: rule.allowedServiceAccounts.map((sa: any) => (sa.resourcePath)) || [],
                        allowedUsers: rule.allowedUsers.map((user: any) => (user.username)) || [],
                        allowedTeams: rule.allowedTeams.map((team: any) => (team.name)) || [],
                        moduleAttestationPolicies: rule.moduleAttestationPolicies.map((att: any) => ({...att, predicateType: att.predicateType === '' ? undefined : att.predicateType}))
                    })),
                    groupPath: group.fullPath
                },
                connections: GetConnections(group.id)
            },
            onCompleted: data => {
                if (data.createManagedIdentity.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createManagedIdentity.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createManagedIdentity.managedIdentity) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../${data.createManagedIdentity.managedIdentity.id}`);
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        })
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: "managed identities", path: 'managed_identities' },
                    { title: "new", path: 'new' }
                ]}
            />
            <Typography variant="h5">New Managed Identity</Typography>
            <ManagedIdentityForm
                groupPath={group.fullPath}
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider light sx={{ marginTop: 4 }} />
            <Box marginTop={2}>
                <LoadingButton
                    loading={isInFlight}
                    disabled={!formData.type}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onCreate}>
                    Create Managed Identity
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default NewManagedIdentity
