import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import GroupPackageForm, { FormData } from './GroupPackageForm';
import { GroupNewPackageFragment_group$key } from './__generated__/GroupNewPackageFragment_group.graphql';
import { GroupNewPackageMutation } from './__generated__/GroupNewPackageMutation.graphql';

interface Props {
    fragmentRef: GroupNewPackageFragment_group$key
}

function GroupNewPackage(props: Props) {
    const navigate = useNavigate();

    const group = useFragment<GroupNewPackageFragment_group$key>(
        graphql`
        fragment GroupNewPackageFragment_group on Group
        {
            id
            fullPath
        }
        `, props.fragmentRef
    );

    const [commit, isInFlight] = useMutation<GroupNewPackageMutation>(graphql`
        mutation GroupNewPackageMutation($input: CreatePackageInput!) {
            createPackage(input: $input) {
                package {
                    id
                    name
                    description
                    visibility
                    allowMutableVersions
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [error, setError] = useState<MutationError>();
    const [formData, setFormData] = useState<FormData>({ name: '', description: '', kind: '', visibility: 'PRIVATE', allowMutableVersions: false });

    const onCreate = () => {
        // Type is required; the Create button is disabled until it is chosen, but guard here too so
        // the empty sentinel is never sent and the type narrows to a valid PackageKind.
        if (!formData.kind) return;
        commit({
            variables: {
                input: {
                    groupId: group.id,
                    name: formData.name,
                    description: formData.description,
                    kind: formData.kind,
                    visibility: formData.visibility,
                    allowMutableVersions: formData.allowMutableVersions,
                }
            },
            onCompleted: data => {
                if (data.createPackage.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createPackage.problems.map(p => p.message).join('; ')
                    });
                } else if (!data.createPackage.package) {
                    setError({ severity: 'error', message: 'Unexpected error occurred' });
                } else {
                    navigate('..');
                }
            },
            onError: err => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${err.message}` });
            }
        });
    };

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: 'packages', path: 'packages' },
                    { title: 'new', path: 'new' },
                ]}
            />
            <Typography variant="h5">New Package</Typography>
            <GroupPackageForm
                data={formData}
                onChange={setFormData}
                error={error}
            />
            <Divider sx={{ marginTop: 4 }} />
            <Box marginTop={2}>
                <Button
                    loading={isInFlight}
                    disabled={!formData.name.trim() || !formData.kind}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onCreate}
                >
                    Create Package
                </Button>
                <Button component={RouterLink} color="inherit" to="..">Cancel</Button>
            </Box>
        </Box>
    );
}

export default GroupNewPackage;
