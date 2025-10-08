import { Box, List, ListItem, ListItemText, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { TerraformModuleVersionDocsDataSourcesFragment_dataResources$key } from './__generated__/TerraformModuleVersionDocsDataSourcesFragment_dataResources.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionDocsDataSourcesFragment_dataResources$key
}

function TerraformModuleVersionDocsDataSources({ fragmentRef }: Props) {
    const data = useFragment<TerraformModuleVersionDocsDataSourcesFragment_dataResources$key>(
        graphql`
            fragment TerraformModuleVersionDocsDataSourcesFragment_dataResources on TerraformModuleConfigurationDetails {
                dataResources {
                    name
                    type
                }
            }
        `, fragmentRef
    );

    return (
        <Box>
            {data.dataResources.length === 0 && <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                <Typography color="textSecondary">This module does not use any datasources</Typography>
            </Box>}
            {data.dataResources.length > 0 && <>
                <Typography color="textSecondary" variant="body1" mb={1}>
                    This module uses the following data sources:
                </Typography>
                <List disablePadding>
                    {data.dataResources.map((source) => (
                        <ListItem key={`${source.type}.${source.name}`} disableGutters>
                            <ListItemText
                                primary={
                                    <SyntaxHighlighter
                                        wrapLines
                                        customStyle={{
                                            fontSize: 14,
                                            marginBottom: -4
                                        }}
                                        language="hcl" style={prismTheme}
                                        children={`data "${source.type}" "${source.name}" {}`}
                                    />
                                }
                            />
                        </ListItem>
                    ))}
                </List>
            </>}
        </Box>
    );
}

export default TerraformModuleVersionDocsDataSources;
