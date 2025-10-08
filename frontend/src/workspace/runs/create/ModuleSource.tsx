import { useState } from "react";
import { Box, Divider, Stack, TextField, Typography } from "@mui/material";
import PanelButton from "../../../common/PanelButton";
import { StyledCode } from "../../../common/StyledCode";
import ModuleAutocomplete, { ModuleOption } from "./ModuleAutocomplete";
import ModuleVersionAutocomplete from "./ModuleVersionAutocomplete";
import { useFragment } from "react-relay/hooks";
import graphql from 'babel-plugin-relay/macro';
import { ModuleSourceFragment_workspace$key } from "./__generated__/ModuleSourceFragment_workspace.graphql";

export type ModuleRunDataOptions = {
    moduleSource: string;
    moduleRegistryType: string;
    moduleVersion: string | null;
};

export const DefaultModuleRunDataOptions: ModuleRunDataOptions = {
    moduleSource: '',
    moduleRegistryType: '',
    moduleVersion: ''
};

interface Props {
    data: ModuleRunDataOptions
    onChange: (data: ModuleRunDataOptions) => void
    fragmentRef: ModuleSourceFragment_workspace$key
}

const ModuleSourceTypes = [
    { name: "Tharsis", title: "Tharsis" },
    { name: "External", title: "Tharsis" }
];

function ModuleSource({ data, onChange, fragmentRef }: Props) {
    const [registryNamespace, setRegistryNamespace] = useState('');
    const [moduleName, setModuleName] = useState('');
    const [system, setSystem] = useState('');

    const workspace = useFragment<ModuleSourceFragment_workspace$key>(
        graphql`
        fragment ModuleSourceFragment_workspace on Workspace
        {
            fullPath
        }
    `, fragmentRef);

    return (
        <Box marginBottom={4}>
            <Typography variant="subtitle1" gutterBottom>Module Registry</Typography>
            <Divider light />
            <Stack sx={{ mt: 2, mb: 4 }} direction="row" spacing={2}>
                {ModuleSourceTypes.map(type => <PanelButton
                    key={type.name}
                    selected={data.moduleRegistryType === type.name}
                    onClick={() => onChange({
                        ...data,
                        moduleRegistryType: type.name, moduleSource: '', moduleVersion: ''
                    }
                    )}
                >
                    <Typography variant="subtitle1">{type.name}</Typography>
                </PanelButton>)}
            </Stack>
            {data.moduleRegistryType && <Box>
                <Typography variant="subtitle1" gutterBottom>Select Module Source</Typography>
                <Divider light sx={{ mb: 2 }} />
            </Box>}
            {data.moduleRegistryType === "Tharsis" && <Box>
                <Box marginBottom={2}>
                    <ModuleAutocomplete
                        workspacePath={workspace.fullPath}
                        onSelected={(value: ModuleOption | null) => {
                            onChange({
                                ...data,
                                moduleVersion: null,
                                moduleSource: value?.source || ''
                            })
                            setRegistryNamespace(value?.registryNamespace || '');
                            setModuleName(value?.name || '');
                            setSystem(value?.system || '')
                        }} />
                    <Typography color="textSecondary" variant="caption">Select one of the Terraform modules available within Tharsis</Typography>
                </Box>
                {data.moduleSource && <Box>
                    <ModuleVersionAutocomplete
                        registryNamespace={registryNamespace}
                        moduleName={moduleName}
                        system={system}
                        value={data.moduleVersion}
                        onSelected={(value: string | null) => onChange({
                            ...data,
                            moduleVersion: value
                        })}
                    />
                    <Typography sx={{ mt: 1 }} color="textSecondary" variant="caption">Select a version for this module. If none is selected, the latest version of the module will be selected by default.</Typography>
                </Box>}
            </Box>}
            {data.moduleRegistryType === "External" && <Box>
                <Box marginBottom={2}>
                    <TextField
                        autoComplete="off"
                        size="small"
                        fullWidth
                        label="Module"
                        value={data.moduleSource}
                        onChange={event => onChange({ ...data, moduleSource: event.target.value })} />
                    <Typography marginTop={1} color="textSecondary" variant="caption">Enter the path to a Terraform module, for example: <StyledCode>example.com/path/to/module_source</StyledCode>
                    </Typography>
                </Box>
                <Stack>
                    <TextField
                        sx={{ maxWidth: 300 }}
                        autoComplete="off"
                        size="small"
                        label="Version"
                        value={data.moduleVersion}
                        onChange={event => onChange({ ...data, moduleVersion: event.target.value })}
                    />
                    <Typography marginTop={1} color="textSecondary" variant="caption">OPTIONAL: Enter a version number for this module, for example: <StyledCode>0.3.0</StyledCode></Typography>
                </Stack>
            </Box>}
        </Box>
    );
}

export default ModuleSource
