import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useFragment, useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import { MutationError } from '../../common/error';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import TerraformModuleForm, { FormData } from './TerraformModuleForm';
import { EditTerraformModuleFragment_group$key } from './__generated__/EditTerraformModuleFragment_group.graphql';
import { EditTerraformModuleMutation } from './__generated__/EditTerraformModuleMutation.graphql';
import { EditTerraformModuleQuery } from './__generated__/EditTerraformModuleQuery.graphql';

interface Props {
    fragmentRef: EditTerraformModuleFragment_group$key
}

function EditTerraformModule(props: Props) {
    const { name, system } = useParams();
    const navigate = useNavigate();

    const group = useFragment<EditTerraformModuleFragment_group$key>(
        graphql`
        fragment EditTerraformModuleFragment_group on Group
        {
            id
            fullPath
        }
        `, props.fragmentRef
    );

    const queryData = useLazyLoadQuery<EditTerraformModuleQuery>(graphql`
        query EditTerraformModuleQuery($registryNamespace: String!, $moduleName: String!, $system: String!) {
            terraformModule(registryNamespace: $registryNamespace, moduleName: $moduleName, system: $system) {
                id
                name
                system
                private
                labels {
                    key
                    value
                }
            }
        }
    `, { registryNamespace: group.fullPath, moduleName: name as string, system: system as string });

    const [commit, isInFlight] = useMutation<EditTerraformModuleMutation>(graphql`
        mutation EditTerraformModuleMutation($input: UpdateTerraformModuleInput!) {
            updateTerraformModule(input: $input) {
                module {
                    id
                    private
                    labels {
                        key
                        value
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

    const module = queryData.terraformModule ?? null;

    const [error, setError] = useState<MutationError>();
    const [formData, setFormData] = useState<FormData | null>(() => module ? {
        private: module.private,
        labels: module.labels.map(l => ({ key: l.key, value: l.value })),
    } : null);

    const onUpdate = () => {
        if (!formData || !module) return;
        commit({
            variables: {
                input: {
                    id: module.id,
                    private: formData.private,
                    labels: formData.labels.filter(l => l.key.trim()).map(l => ({ key: l.key, value: l.value })),
                }
            },
            onCompleted: data => {
                if (data.updateTerraformModule.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updateTerraformModule.problems.map((p: any) => p.message).join('; ')
                    });
                } else if (!data.updateTerraformModule.module) {
                    setError({ severity: 'error', message: 'Unexpected error occurred' });
                } else {
                    navigate(-1);
                }
            },
            onError: err => {
                setError({ severity: 'error', message: `Unexpected error occurred: ${err.message}` });
            }
        });
    };

    return formData && module ? (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={group.fullPath}
                childRoutes={[
                    { title: 'terraform modules', path: 'terraform_modules' },
                    { title: `${module.name}/${module.system}`, path: `${module.name}/${module.system}` },
                    { title: 'edit', path: 'edit' },
                ]}
            />
            <Typography variant="h5">Edit Terraform Module</Typography>
            <TerraformModuleForm
                editMode
                data={formData}
                onChange={setFormData}
                error={error}
            />
            <Divider sx={{ marginTop: 4 }} />
            <Box marginTop={2}>
                <Button
                    loading={isInFlight}
                    variant="outlined"
                    color="primary"
                    sx={{ marginRight: 2 }}
                    onClick={onUpdate}
                >
                    Update Module
                </Button>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    ) : <Box>Terraform Module not found</Box>;
}

export default EditTerraformModule;
