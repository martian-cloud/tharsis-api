import { Box, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import remarkGfm from 'remark-gfm';
import MuiMarkdown from '../../common/Markdown';
import { TerraformModuleVersionDocsFragment_configurationDetails$key } from './__generated__/TerraformModuleVersionDocsFragment_configurationDetails.graphql';
import TerraformModuleVersionDocsDataSources from './TerraformModuleVersionDocsDataSources';
import TerraformModuleVersionDocsInputs from './TerraformModuleVersionDocsInputs';
import TerraformModuleVersionDocsOutputs from './TerraformModuleVersionDocsOutputs';
import TerraformModuleVersionDocsRequiredProviders from './TerraformModuleVersionDocsRequiredProviders';
import TerraformModuleVersionDocsResources from './TerraformModuleVersionDocsResources';
import TerraformModuleVersionDocsSidebar from './TerraformModuleVersionDocsSidebar';

interface Props {
    fragmentRef: TerraformModuleVersionDocsFragment_configurationDetails$key | null | undefined
}

function TerraformModuleVersionDocs({ fragmentRef }: Props) {
    const theme = useTheme();
    const [searchParams, setSearchParams] = useSearchParams();

    const data = useFragment<TerraformModuleVersionDocsFragment_configurationDetails$key>(
        graphql`
            fragment TerraformModuleVersionDocsFragment_configurationDetails on TerraformModuleConfigurationDetails {
                readme
                ...TerraformModuleVersionDocsSidebarFragment_configurationDetails
                ...TerraformModuleVersionDocsInputsFragment_variables
                ...TerraformModuleVersionDocsOutputsFragment_outputs
                ...TerraformModuleVersionDocsResourcesFragment_managedResources
                ...TerraformModuleVersionDocsDataSourcesFragment_dataResources
                ...TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders
            }
        `, fragmentRef
    );

    const onTreeItemChange = (item: string) => {
        searchParams.set('item', item);
        setSearchParams(searchParams, { replace: true });
    };

    const selected = useMemo(() => {
        const response = searchParams.get('item');
        return response ? response : (data?.readme ? 'overview' : 'inputs');
    }, [data, searchParams]);

    if (!data) {
        return (
            <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                <Typography color="textSecondary">No documentation for this version</Typography>
            </Box>
        );
    }

    return (
        <Box
            display="flex"
            width="100%"
            sx={{
                [theme.breakpoints.down('lg')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *': { mb: 4 },
                }
            }}
        >
            <TerraformModuleVersionDocsSidebar fragmentRef={data} onItemChange={onTreeItemChange} />
            <Box
                flex={1}
                pl={2}
                width="75%"
                borderLeft={`1px solid ${theme.palette.divider}`}
                sx={{
                    [theme.breakpoints.down('lg')]: {
                        borderLeft: 'none',
                        paddingLeft: 0,
                        width: '100%'
                    }
                }}
            >
                {selected === 'overview' && (
                    <Box>
                        <MuiMarkdown
                            children={data.readme}
                            remarkPlugins={[remarkGfm]}
                        />
                    </Box>
                )}
                {selected === 'inputs' && <TerraformModuleVersionDocsInputs fragmentRef={data} />}
                {selected === 'outputs' && <TerraformModuleVersionDocsOutputs fragmentRef={data} />}
                {selected === 'resources' && <TerraformModuleVersionDocsResources fragmentRef={data} />}
                {selected === 'dataSources' && <TerraformModuleVersionDocsDataSources fragmentRef={data} />}
                {selected === 'requiredProviders' && <TerraformModuleVersionDocsRequiredProviders fragmentRef={data} />}
            </Box>
        </Box>
    );
}

export default TerraformModuleVersionDocs;
