import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import GroupPackageForm, { FormData, PackageKind, PackageVisibility } from './GroupPackageForm';
import { GroupEditPackageFragment_group$key } from './__generated__/GroupEditPackageFragment_group.graphql';
import { GroupEditPackageMutation } from './__generated__/GroupEditPackageMutation.graphql';
import { GroupEditPackageDeleteMutation } from './__generated__/GroupEditPackageDeleteMutation.graphql';
import { GroupEditPackageQuery } from './__generated__/GroupEditPackageQuery.graphql';

interface Props {
    fragmentRef: GroupEditPackageFragment_group$key
}

function GroupEditPackage(props: Props) {
    const { id } = useParams();
    const navigate = useNavigate();

    const group = useFragment<GroupEditPackageFragment_group$key>(
        graphql`
        fragment GroupEditPackageFragment_group on Group
        {
            id
            fullPath
        }
        `, props.fragmentRef
    );

    const queryData = useLazyLoadQuery<GroupEditPackageQuery>(graphql`
        query GroupEditPackageQuery($id: String!) {
            node(id: $id) {
                ... on Package {
                    id
                    name
                    description
                    kind
                    visibility
                    allowMutableVersions
                }
            }
        }
    `, { id: id as string });

    const [commit, isInFlight] = useMutation<GroupEditPackageMutation>(graphql`
        mutation GroupEditPackageMutation($input: UpdatePackageInput!) {
            updatePackage(input: $input) {
                package {
                    id
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

    const [commitDelete, isDeleting] = useMutation<GroupEditPackageDeleteMutation>(graphql`
        mutation GroupEditPackageDeleteMutation($input: DeletePackageInput!) {
            deletePackage(input: $input) {
                package {
                    id
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const pkg = queryData.node?.name !== undefined ? queryData.node : null;

    const [error, setError] = useState<MutationError>();
    const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
    const [formData, setFormData] = useState<FormData | null>(() => pkg ? {
        name: pkg.name ?? '',
        description: pkg.description ?? '',
        kind: pkg.kind as PackageKind,
        visibility: pkg.visibility as PackageVisibility,
        allowMutableVersions: pkg.allowMutableVersions ?? false,
    } : null);

    const listPath = `/groups/${group.fullPath}/-/packages`;

    const onUpdate = () => {
        if (!formData || !pkg) return;
        commit({
            variables: {
                input: {
                    id: pkg.id!,
                    description: formData.description,
                    visibility: formData.visibility,
                    allowMutableVersions: formData.allowMutableVersions,
                }
            },
            onCompleted: data => {
                if (data.updatePackage.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updatePackage.problems.map(p => p.message).join('; ')
                    });
                } else if (!data.updatePackage.package) {
                    setError({ severity: 'error', message: 'Unexpected error occurred' });
                } else {
                    navigate(listPath);
                }
            },
            onError: err => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${err.message}` });
            }
        });
    };

    const onDeleteConfirm = (confirm?: boolean) => {
        if (!confirm || !pkg) {
            setShowDeleteConfirm(false);
            return;
        }
        commitDelete({
            variables: { input: { id: pkg.id! } },
            onCompleted: data => {
                setShowDeleteConfirm(false);
                if (data.deletePackage.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.deletePackage.problems.map(p => p.message).join('; ')
                    });
                } else {
                    navigate(listPath);
                }
            },
            onError: err => {
                setShowDeleteConfirm(false);
                setError({ severity: 'error', message: `Unexpected error occurred: ${err.message}` });
            }
        });
    };

    return formData && pkg ? (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: 'packages', path: 'packages' },
                    { title: pkg.name!, path: `${pkg.id!}` },
                    { title: 'edit', path: 'edit' },
                ]}
            />
            <Typography variant="h5">Edit Package</Typography>
            <GroupPackageForm
                editMode
                data={formData}
                onChange={setFormData}
                error={error}
            />
            <Divider sx={{ marginTop: 4 }} />
            <Box marginTop={2} display="flex" alignItems="center">
                <Button
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onUpdate}
                >
                    Update Package
                </Button>
                <Button component={RouterLink} color="inherit" to={listPath}>Cancel</Button>
                <Box flex={1} />
                <Button color="error" onClick={() => setShowDeleteConfirm(true)}>Delete</Button>
            </Box>
            {showDeleteConfirm && (
                <ConfirmationDialog
                    title="Delete Package"
                    confirmLabel="Delete"
                    confirmInProgress={isDeleting}
                    onConfirm={() => onDeleteConfirm(true)}
                    onClose={() => onDeleteConfirm()}
                >
                    Are you sure you want to delete package <strong>{pkg.name}</strong>?
                </ConfirmationDialog>
            )}
        </Box>
    ) : <Box>Package not found</Box>;
}

export default GroupEditPackage;
