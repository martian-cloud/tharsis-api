import { Box, Chip, List, ListItem, ListItemText, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { TerraformModuleVersionDocsOutputsFragment_outputs$key } from './__generated__/TerraformModuleVersionDocsOutputsFragment_outputs.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionDocsOutputsFragment_outputs$key
}

function TerraformModuleVersionDocsOutputs({ fragmentRef }: Props) {
    const data = useFragment<TerraformModuleVersionDocsOutputsFragment_outputs$key>(
        graphql`
            fragment TerraformModuleVersionDocsOutputsFragment_outputs on TerraformModuleConfigurationDetails {
                outputs {
                    name
                    description
                    sensitive
                }
            }
        `, fragmentRef
    );

    return (
        <Box>
            {data.outputs.length === 0 && <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                <Typography color="textSecondary">This module does not provide any outputs</Typography>
            </Box>}
            {data.outputs.length > 0 && <>
                <Typography color="textSecondary" variant="body1" mb={2}>
                    This module provides the following outputs:
                </Typography>
                <List disablePadding>
                    {data.outputs.map((output, index) => (
                        <ListItem key={output.name} divider={index !== (data.outputs.length - 1)}>
                            <ListItemText
                                primary={<Box display="flex" alignItems="center">
                                    <Typography sx={{ fontWeight: 'bold' }} component="code">{output.name}</Typography>
                                    {output.sensitive && <Chip
                                        sx={{ ml: 1 }}
                                        size="xs"
                                        variant="outlined"
                                        color="warning"
                                        label='sensitive'
                                    />}
                                </Box>}
                                secondary={output.description}
                            />
                        </ListItem>
                    ))}
                </List>
            </>}
        </Box>
    );
}

export default TerraformModuleVersionDocsOutputs;
