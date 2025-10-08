import { Alert, Box, Button, Chip, FormControlLabel, Paper, Switch, TextField, Typography } from "@mui/material";
import graphql from 'babel-plugin-relay/macro';
import { useState } from "react";
import { useFragment } from "react-relay";
import { MutationError } from "../../../common/error";
import { StyledCode } from "../../../common/StyledCode";
import VCSProviderAutocomplete, { VCSProviderOption } from "./VCSProviderAutocomplete";
import { VCSProviderLinkFormFragment_workspace$key } from "./__generated__/VCSProviderLinkFormFragment_workspace.graphql";

export interface VCSFormData {
    id: string
    repositoryPath: string
    moduleDirectory: string
    branch: string | null
    tagRegex: string | null
    globPatterns: readonly string[] | string[]
    autoSpeculativePlan: boolean
    webhookDisabled: boolean
    label: string
    description: string
    type: string | undefined
}

interface Props {
    viewMode?: boolean
    data: VCSFormData
    onChange: (data: VCSFormData) => void
    fragmentRef: VCSProviderLinkFormFragment_workspace$key
    error?: MutationError
}

function VCSProviderLinkForm(props: Props) {
    const { viewMode, data, onChange, error } = props

    const [globToAdd, setGlobToAdd] = useState<string>('')
    const [disableRepoPath, setDisableRepoPath] = useState<boolean>(viewMode ? true : false)

    const vcsProviderData = useFragment<VCSProviderLinkFormFragment_workspace$key>(
        graphql`
        fragment VCSProviderLinkFormFragment_workspace on Workspace
        {
            fullPath
            workspaceVcsProviderLink {
                id
                repositoryPath
                branch
                moduleDirectory
                tagRegex
                globPatterns
                autoSpeculativePlan
                webhookDisabled
                vcsProvider {
                    id
                    name
                    description
                    type
                    autoCreateWebhooks
                }
            }
        }
        `, props.fragmentRef
    )

    const [selected, setSelected] = useState<VCSProviderOption | null>(vcsProviderData?.workspaceVcsProviderLink ? {
        id: vcsProviderData?.workspaceVcsProviderLink?.vcsProvider.id || '',
        label: vcsProviderData?.workspaceVcsProviderLink?.vcsProvider.name || '',
        description: vcsProviderData?.workspaceVcsProviderLink?.vcsProvider.description || '',
        type: vcsProviderData?.workspaceVcsProviderLink?.vcsProvider.type || '',
    } : null);

    const addGlob = () => {
        onChange({ ...data, globPatterns: [...data.globPatterns, globToAdd] })
        setGlobToAdd('')
    }

    const deleteGlob = (globToDelete: string) => () => {
        onChange({ ...data, globPatterns: data.globPatterns.filter((gl) => gl !== globToDelete) })
    };

    const onVCSProviderSelected = (value: any) => {
        const clearFormObj: VCSFormData = {
            id: '',
            repositoryPath: '',
            moduleDirectory: '',
            branch: '',
            tagRegex: '',
            globPatterns: [],
            autoSpeculativePlan: false,
            webhookDisabled: false,
            label: '',
            description: '',
            type: ''
        }

        if (value) {
            if (value.label === vcsProviderData?.workspaceVcsProviderLink?.vcsProvider.name) {
                const vcsProvider = vcsProviderData?.workspaceVcsProviderLink as any
                setDisableRepoPath(true)
                onChange({
                    id: value.id,
                    repositoryPath: vcsProvider.repositoryPath,
                    moduleDirectory: vcsProvider.moduleDirectory,
                    branch: vcsProvider.branch,
                    tagRegex: vcsProvider.tagRegex,
                    globPatterns: vcsProvider.globPatterns,
                    autoSpeculativePlan: vcsProvider.autoSpeculativePlan,
                    webhookDisabled: vcsProvider.webhookDisabled,
                    label: vcsProvider.name,
                    description: vcsProvider.description,
                    type: vcsProvider.type
                })
            } else {
                setSelected(value)
                setDisableRepoPath(false)
                onChange({ ...clearFormObj, id: value.id })
            }
        } else {
            setSelected(null)
            onChange(clearFormObj)
            setDisableRepoPath(false)
        }
    };

    return (
        <Box sx={{ marginTop: 3 }}>
            {error && <Alert sx={{ marginTop: 2, marginBottom: 4 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Box sx={{ mb: 4 }}>
                <VCSProviderAutocomplete
                    path={vcsProviderData.fullPath}
                    value={selected}
                    onSelected={(value: any) => onVCSProviderSelected(value)}
                />
                <Typography sx={{ mb: 4 }} variant="subtitle2">VCS providers within this group and inherited from parent groups</Typography>
            </Box>
            <Box sx={{ marginTop: 2, marginBottom: 2 }}>
                <TextField
                    sx={{ mb: 1 }}
                    disabled={disableRepoPath}
                    size="small"
                    margin="dense"
                    fullWidth
                    label="Repository Path"
                    value={data.repositoryPath}
                    onChange={event => onChange({ ...data, repositoryPath: event.target.value })}
                />
                <Typography sx={{ mb: 4 }} variant="subtitle2">Enter path to repository, &#x28;e.g., <StyledCode>firstname_lastname/name_of_repository</StyledCode>&#x29;</Typography>
                <TextField
                    sx={{ mb: 1 }}
                    size="small"
                    margin="dense"
                    fullWidth
                    label="Branch"
                    value={data.branch}
                    onChange={event => onChange({ ...data, branch: event.target.value })}
                />
                <Typography sx={{ mb: 4 }} variant="subtitle2">Branch to where this workspace will connect &#x28;e.g., <StyledCode>main</StyledCode>&#x29;. If no branch is entered, the workspace will default to the main branch of the repository.</Typography>
                <TextField
                    sx={{ mb: 1 }}
                    size="small"
                    margin="dense"
                    fullWidth
                    label="Module Directory"
                    value={data.moduleDirectory}
                    onChange={event => onChange({ ...data, moduleDirectory: event.target.value })}
                />
                <Typography sx={{ mb: 4 }} variant="subtitle2">Relative path to the directory that contains Terraform modules to run &#x28;e.g., <StyledCode>src/modules</StyledCode>&#x29;. If no module directory is listed, the workspace will default to the repository's root module in the base directory. </Typography>
                <TextField
                    sx={{ mb: 1 }}
                    size="small"
                    margin="dense"
                    fullWidth
                    label="Tag Regular Expression"
                    value={data.tagRegex}
                    onChange={event => onChange({ ...data, tagRegex: event.target.value })}
                />
                <Typography sx={{ mb: 4 }} variant="subtitle2">A tag regular expression defines the commit tag format that may create a Tharsis run. For example, the regular expression <StyledCode>\d+.\d+.\d+$</StyledCode> only allows tags like <StyledCode>v0.0.1</StyledCode> to create runs. If no tag regular expression is defined, then all tagged commits are ignored.</Typography>
                <Box display="flex" sx={{ mb: 1 }}>
                    <TextField
                        size="small"
                        margin="dense"
                        label="Glob Patterns"
                        value={globToAdd}
                        onChange={event => setGlobToAdd(event.target.value)}
                    />
                    <Button
                        disabled={!globToAdd}
                        onClick={addGlob}
                    >Add</Button>
                </Box>
                <Box sx={{ mb: data.globPatterns.length > 0 ? 1 : 4 }}>
                    <Typography
                        marginBottom={1}
                        variant="subtitle2">Glob patterns are triggers for automatic Tharsis runs. When defined, the workspace only creates runs when certain files or directories that have changed in a commit match the glob pattern(s). If any pattern matches, a run will be triggered.
                    </Typography>
                    <Typography variant="subtitle2">You can generate a list of glob patterns to add to your VCS provider link by entering a glob pattern in the field above and clicking <StyledCode>ADD</StyledCode>.
                    </Typography>
                </Box>

                {data.globPatterns.length > 0 &&
                    <Paper
                        sx={{ mb: 4, p: 0.5, minHeight: 50 }}>
                        {data.globPatterns.map((glob, idx) =>
                            <Chip
                                sx={{ m: 0.25 }}
                                key={idx}
                                label={glob}
                                onDelete={deleteGlob(glob)}
                            />)}
                    </Paper>}
                <Box sx={{ mb: 4 }}>
                    <Typography variant="subtitle1">Allow speculative run for pull and merge requests</Typography>
                    <FormControlLabel
                        control={<Switch
                            sx={{ m: 2 }}
                            checked={data.autoSpeculativePlan}
                            color="secondary"
                            onChange={event => onChange({ ...data, autoSpeculativePlan: event.target.checked })}
                        />}
                        label={data.autoSpeculativePlan ? 'On' : 'Off'}
                    />
                    <Typography variant="subtitle2">Selecting this allows Tharsis to automatically create speculative plans. Speculative plans are plan-only runs, which show a set of possible changes. Apply plans will be not created. Pull and merge requests outside of {data.repositoryPath || 'the specified repository'} are ignored.</Typography>
                </Box>
                <Box sx={{ mb: 4 }}>
                    <Typography variant="subtitle1">Disable webhooks?</Typography>
                    <FormControlLabel
                        control={<Switch
                            sx={{ m: 2 }}
                            checked={data.webhookDisabled}
                            color="secondary"
                            onChange={event => onChange({ ...data, webhookDisabled: event.target.checked })}
                        />}
                        label={data.webhookDisabled ? 'On' : 'Off'}
                    />
                    <Typography variant="subtitle2">When <StyledCode>On</StyledCode>, all webhook events are ignored. VCS runs can still be created manually.</Typography>
                </Box>
            </Box>
        </Box>
    )
}

export default VCSProviderLinkForm
