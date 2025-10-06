import { Alert, Box, Divider, Stack, Typography } from "@mui/material";
import { MutationError } from '../../../common/error';
import PanelButton from "../../../common/PanelButton";
import ConfigurationVersionSource, { ConfigVersionRunDataOptions, DefaultConfigVersionRunDataOptions } from "./ConfigurationVersionSource";
import ModuleSource, { DefaultModuleRunDataOptions, ModuleRunDataOptions } from "./ModuleSource";
import VCSWorkspaceLinkSource, { DefaultVCSRunDataOptions, VCSRunDataOptions } from "./VCSWorkspaceLinkSource";
import { ModuleSourceFragment_workspace$key } from "./__generated__/ModuleSourceFragment_workspace.graphql";
import { VCSWorkspaceLinkSourceFragment_workspace$key } from "./__generated__/VCSWorkspaceLinkSourceFragment_workspace.graphql";

export type RunFormData = {
    source: string;
    runType: string | null;
    options: ModuleRunDataOptions | ConfigVersionRunDataOptions | VCSRunDataOptions | null;
};

type FragmentRef = VCSWorkspaceLinkSourceFragment_workspace$key | ModuleSourceFragment_workspace$key;

interface Props {
    error?: MutationError
    data: RunFormData
    onChange: (data: RunFormData) => void
    fragmentRef: FragmentRef
}

const SOURCE_TYPES = [
    { name: 'module', title: 'Module', description: 'Run will be created from a module within a Terraform Module Registry' },
    { name: 'configuration_version', title: 'Configuration Version', description: 'Run will be created using an uploaded module tar' },
    { name: 'vcs', title: 'VCS Workspace Link', description: 'Run will be created via this workspace\'s link to the VCS Provider' }
];

const RUN_TYPES = [
    { label: 'Plan', type: 'plan', description: 'Create a speculative plan (a plan-only run), which shows a possible set of changes. It cannot be applied.' },
    { label: 'Apply', type: 'apply', description: 'After a plan run is successfully created, any changes can be manually applied.' }
];

const DEFAULT_RUN_OPTIONS: { [key: string]: ModuleRunDataOptions | ConfigVersionRunDataOptions | VCSRunDataOptions } = {
    module: DefaultModuleRunDataOptions,
    configuration_version: DefaultConfigVersionRunDataOptions,
    vcs: DefaultVCSRunDataOptions
};

function CreateRunForm({ error, data, onChange, fragmentRef }: Props) {
    return (
        <Box>
            {error && <Alert sx={{ marginTop: 2, mb: 2 }} severity={error.severity}>
                {error.message}
            </Alert>}
            <Box sx={{ mt: 4, mb: 4 }}>
                <Typography variant="subtitle1" gutterBottom>Select Source</Typography>
                <Divider light />
                <Stack marginTop={2} direction="row" spacing={2}>
                    {SOURCE_TYPES.map(source => <PanelButton
                        key={source.name}
                        selected={data.source === source.name}
                        onClick={() => {
                            onChange({
                                source: source.name,
                                runType: '',
                                options: DEFAULT_RUN_OPTIONS[source.name]
                            })
                        }}>
                        <Typography variant="subtitle1" align="center">{source.title}</Typography>
                        <Typography variant="caption" align="center">
                            {source.description}
                        </Typography>
                    </PanelButton>)}
                </Stack>
            </Box>
            {(data.source === 'module' || data.source === 'configuration_version') && <Box marginBottom={4}>
                <Typography variant="subtitle1" gutterBottom>Select Run Type</Typography>
                <Divider light />
                <Stack marginTop={2} marginBottom={2} direction="row" spacing={2}>
                    {RUN_TYPES.map(run => <PanelButton
                        key={run.type}
                        selected={data.runType === run.type}
                        onClick={() => onChange({ ...data, runType: run.type })}
                    >
                        <Typography variant="subtitle1">{run.label}</Typography>
                        <Typography variant="caption" align="center">
                            {run.description}
                        </Typography>
                    </PanelButton>)}
                </Stack>
            </Box>}
            {data.source === 'module' &&
                <ModuleSource
                    data={data.options as ModuleRunDataOptions}
                    onChange={(options: ModuleRunDataOptions) => onChange({ ...data, options })}
                    fragmentRef={fragmentRef as ModuleSourceFragment_workspace$key} />
            }
            {data.source === 'configuration_version' &&
                <ConfigurationVersionSource
                    data={data.options as ConfigVersionRunDataOptions}
                    onChange={(options: ConfigVersionRunDataOptions) => onChange({ ...data, options })} />
            }
            {data.source === 'vcs' &&
                <VCSWorkspaceLinkSource
                    data={data.options as VCSRunDataOptions}
                    onChange={(options: VCSRunDataOptions) => onChange({ ...data, options })}
                    fragmentRef={fragmentRef as VCSWorkspaceLinkSourceFragment_workspace$key} />
            }
            <Divider light />
        </Box>
    );
}

export default CreateRunForm
