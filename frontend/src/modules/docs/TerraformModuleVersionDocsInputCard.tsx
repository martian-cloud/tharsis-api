import { Box, Card, CardContent, CardHeader, Chip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { TerraformModuleVersionDocsInputCardFragment_variable$key } from './__generated__/TerraformModuleVersionDocsInputCardFragment_variable.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionDocsInputCardFragment_variable$key
}

function TerraformModuleVersionDocsInputCard({ fragmentRef }: Props) {

    const variable = useFragment<TerraformModuleVersionDocsInputCardFragment_variable$key>(
        graphql`
            fragment TerraformModuleVersionDocsInputCardFragment_variable on TerraformModuleConfigurationDetailsVariable {
                    name
                    type
                    description
                    default
                    required
                    sensitive
            }
        `, fragmentRef
    );

    return (
        <Card variant="outlined" sx={{ mb: 2 }}>
            <CardHeader
                sx={{ pb: 0 }}
                title={
                    <Box display="flex" alignItems="center" mb={1}>
                        <Typography sx={{ fontWeight: 'bold' }} component="code">{variable.name}</Typography>
                        {variable.sensitive && <Chip
                            sx={{ ml: 1 }}
                            size="xs"
                            variant="outlined"
                            color="warning"
                            label='sensitive'
                        />}
                    </Box>
                }
                subheader={variable.type ? <Box display="flex">
                    <SyntaxHighlighter
                        wrapLines
                        customStyle={{ fontSize: 13, padding: '8px', margin: 0 }}
                        language="hcl"
                        style={prismTheme}
                        children={variable.type}
                    />
                </Box> : <Typography color="textSecondary">No type specified</Typography>}
            />
            <CardContent>
                {variable.description && <Typography mb={1} color="textSecondary">
                    {variable.description}
                </Typography>}
                {!variable.default && <Box display="flex">
                    <Typography mr={1}>Default:</Typography>
                    <Typography color="textSecondary">None</Typography>
                </Box>}
                {variable.default && <Box>
                    <Typography mb={1}>Default:</Typography>
                    <Box display="flex">
                        <SyntaxHighlighter
                            wrapLines
                            customStyle={{ fontSize: 13, padding: '8px', margin: 0 }}
                            language="hcl"
                            style={prismTheme}
                            children={variable.default}
                        />
                    </Box>
                </Box>}
            </CardContent>
        </Card>
    );
}

export default TerraformModuleVersionDocsInputCard;
