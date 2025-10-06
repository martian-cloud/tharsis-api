import { LoadingButton } from "@mui/lab";
import { Box, Button, Typography } from "@mui/material";
import graphql from 'babel-plugin-relay/macro';
import { useContext, useMemo, useState } from "react";
import { GraphQLTaggedNode, useFragment, useMutation } from "react-relay/hooks";
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import AuthServiceContext from "../../../auth/AuthServiceContext";
import AuthenticationService from "../../../auth/AuthenticationService";
import { MutationError } from '../../../common/error';
import NamespaceBreadcrumbs from '../../../namespace/NamespaceBreadcrumbs';
import { ConfigVersionRunDataOptions } from "./ConfigurationVersionSource";
import CreateRunForm, { RunFormData } from "./CreateRunForm";
import { ModuleRunDataOptions } from "./ModuleSource";
import { VCSRunDataOptions } from "./VCSWorkspaceLinkSource";
import { CreateRunFragment_workspace$key } from "./__generated__/CreateRunFragment_workspace.graphql";
import { CreateRun_ConfigRunMutation } from "./__generated__/CreateRun_ConfigRunMutation.graphql";
import { CreateRun_RunMutation } from "./__generated__/CreateRun_RunMutation.graphql";
import { CreateRun_VCSRunMutation } from "./__generated__/CreateRun_VCSRunMutation.graphql";
import { uploadConfigVersionPackage } from "./configVersion";

interface Props {
    fragmentRef: CreateRunFragment_workspace$key
}

export const VCSRunMutation: GraphQLTaggedNode =
    graphql`
    mutation CreateRun_VCSRunMutation($input: CreateVCSRunInput!) {
        createVCSRun (input: $input) {
            problems {
                message
                field
                type
            }
        }
    }`

export const CreateRunMutation: GraphQLTaggedNode =
    graphql`
    mutation CreateRun_RunMutation($input: CreateRunInput!) {
        createRun (input: $input) {
            run {
                id
            }
            problems {
                message
                field
                type
            }
        }
    }`

function CreateRun({ fragmentRef }: Props) {
    const navigate = useNavigate();
    const authService = useContext<AuthenticationService>(AuthServiceContext);
    const [error, setError] = useState<MutationError>();

    const workspace = useFragment<CreateRunFragment_workspace$key>(
        graphql`
        fragment CreateRunFragment_workspace on Workspace
        {
            id
            fullPath
            workspaceVcsProviderLink {
                id
            }
            ...ModuleSourceFragment_workspace
            ...VCSWorkspaceLinkSourceFragment_workspace
        }
    `, fragmentRef);

    const [formData, setFormData] = useState<RunFormData>({ source: '', runType: '', options: null });

    const [commitVCSRun, vcsRunIsInFlight] = useMutation<CreateRun_VCSRunMutation>(VCSRunMutation);
    const [commitModuleRun, moduleRunIsInFlight] = useMutation<CreateRun_RunMutation>(CreateRunMutation);
    const [commitRun, runIsInFlight] = useMutation<CreateRun_RunMutation>(CreateRunMutation);

    const [commitConfigRun] = useMutation<CreateRun_ConfigRunMutation>(graphql`
        mutation CreateRun_ConfigRunMutation($input: CreateConfigurationVersionInput!) {
            createConfigurationVersion(input: $input) {
                configurationVersion {
                    id
                    status
                    workspaceId
                    vcsEvent {
                        type
                        status
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

    const onCreateConfigVersionId = () => {
        const options = formData.options as ConfigVersionRunDataOptions;
        commitConfigRun({
            variables: {
                input: {
                    workspacePath: workspace.fullPath,
                    speculative: formData.runType === 'plan'
                }
            },
            onCompleted: data => {
                if (data.createConfigurationVersion.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createConfigurationVersion.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createConfigurationVersion || !data.createConfigurationVersion.configurationVersion) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    const configVersionId = data.createConfigurationVersion.configurationVersion?.id as string;
                    uploadConfigVersionPackage(options.file, workspace.id, configVersionId, authService)
                        .then(() => createConfigVersionRun(configVersionId))
                        .catch(error => {
                            setError({
                                severity: 'warning',
                                message: `failed to upload configuration version: ${error.message}`
                            });
                        });
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

    const createConfigVersionRun = (configVersionId: string) => {
        commitRun({
            variables: {
                input: {
                    workspacePath: workspace.fullPath,
                    configurationVersionId: configVersionId,
                }
            },
            onCompleted: data => {
                if (data.createRun.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createRun.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createRun) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../`);
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

    const onCreateVCSRun = () => {
        const options = formData.options as VCSRunDataOptions;
        commitVCSRun({
            variables: {
                input: {
                    workspacePath: workspace.fullPath,
                    referenceName: options.referenceName ? options.referenceName.trim() : undefined
                }
            },
            onCompleted: data => {
                if (data.createVCSRun.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createVCSRun.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createVCSRun) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate('../');
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

    const onCreateModuleRun = () => {
        const options = formData.options as ModuleRunDataOptions;
        commitModuleRun({
            variables: {
                input: {
                    workspacePath: workspace.fullPath,
                    moduleSource: options.moduleSource,
                    moduleVersion: options.moduleVersion === '' ? null : options.moduleVersion?.trim(),
                    speculative: formData.runType === 'plan'
                }
            },
            onCompleted: data => {
                if (data.createRun.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createRun.problems.map(problem => problem.message).join('; ')
                    });
                } else if (!data.createRun) {
                    setError({
                        severity: 'error',
                        message: "Unexpected error occurred"
                    });
                } else {
                    navigate(`../${data.createRun.run?.id}`);
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

    const onCreateRun = () => {
        switch (formData.source) {
            case 'module':
                onCreateModuleRun();
                break;
            case 'configuration_version':
                onCreateConfigVersionId();
                break;
            case 'vcs':
                onCreateVCSRun();
        }
    }

    const enableButton = useMemo(() => (formData.source === 'vcs' && workspace.workspaceVcsProviderLink) ||
        (formData.source === 'module' && (formData.options as ModuleRunDataOptions).moduleSource !== '' && formData.runType !== '') ||
        (formData.source === 'configuration_version' && (formData.options as ConfigVersionRunDataOptions).file && formData.runType),
        [formData]);

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={workspace.fullPath}
                childRoutes={[
                    { title: "runs", path: 'runs' },
                    { title: "create", path: 'create' }
                ]}
            />
            <Typography variant="h5">Create Run</Typography>
            <CreateRunForm
                data={formData}
                error={error}
                fragmentRef={workspace}
                onChange={(data: RunFormData) => {
                    setFormData(data)
                    setError(undefined)
                }}
            />
            <Box marginTop={2}>
                <LoadingButton
                    sx={{ marginRight: 2 }}
                    loading={vcsRunIsInFlight || moduleRunIsInFlight || runIsInFlight}
                    disabled={!enableButton}
                    variant="outlined"
                    color="primary"
                    onClick={onCreateRun}
                >
                    Create Run
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>Cancel</Button>
            </Box>
        </Box>
    );
}

export default CreateRun
