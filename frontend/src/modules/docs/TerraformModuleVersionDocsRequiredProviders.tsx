import { Box, List, ListItem, ListItemText, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders$key } from './__generated__/TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders$key
}

function buildProviderInfo(moduleName: string, versionConstraints: readonly string[], source: string) {
    let response = `${moduleName} = {\n   source  = "${source}"\n`;

    if (versionConstraints.length > 0) {
        response += `   version = "${versionConstraints.join(', ')}"\n`;
    }

    response += "}\n";

    return response;
}

const getSystemName = (source: string) => {
    const parts = source.split('/');
    return parts[parts.length - 1];
}

function TerraformModuleVersionDocsRequiredProviders({ fragmentRef }: Props) {
    const data = useFragment<TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders$key>(
        graphql`
            fragment TerraformModuleVersionDocsRequiredProvidersFragment_requiredProviders on TerraformModuleConfigurationDetails {
                requiredProviders {
                source
                versionConstraints
            }
        }
        `, fragmentRef
    );

    const hasProviders = useMemo(() =>
        data.requiredProviders.filter(provider => provider.source !== "").length > 0,
        [data.requiredProviders]
    );

    return (
        <Box>
            {!hasProviders && <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                <Typography color="textSecondary">This module does not require any providers</Typography>
            </Box>}
            {hasProviders && <>
                <Typography color="textSecondary" variant="body1" mb={1}>
                    This module requires the following providers:
                </Typography>
                <List>
                    {data.requiredProviders.map((provider) => (
                        provider.source && (
                            <ListItem key={provider.source}
                                component={Box}
                                disableGutters
                                disablePadding
                            >
                                <ListItemText
                                    primary={
                                        <SyntaxHighlighter
                                            wrapLines
                                            customStyle={{ fontSize: 13 }}
                                            language="hcl"
                                            style={prismTheme}
                                            children={buildProviderInfo(
                                                getSystemName(provider.source),
                                                provider.versionConstraints,
                                                provider.source
                                            )}
                                        />
                                    }
                                />
                            </ListItem>
                        )
                    ))}
                </List>
            </>}
        </Box>
    );
}

export default TerraformModuleVersionDocsRequiredProviders;
