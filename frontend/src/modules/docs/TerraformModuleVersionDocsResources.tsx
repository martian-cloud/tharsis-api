import { Box, List, ListItem, ListItemText, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { StyledCode } from '../../common/StyledCode';
import { TerraformModuleVersionDocsResourcesFragment_managedResources$key } from './__generated__/TerraformModuleVersionDocsResourcesFragment_managedResources.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionDocsResourcesFragment_managedResources$key
}

function TerraformModuleVersionDocsResources({ fragmentRef }: Props) {
    const data = useFragment<TerraformModuleVersionDocsResourcesFragment_managedResources$key>(
        graphql`
            fragment TerraformModuleVersionDocsResourcesFragment_managedResources on TerraformModuleConfigurationDetails {
                managedResources {
                    name
                    type
                }
            }
        `, fragmentRef
    );

    return (
        <Box>
            {data.managedResources.length === 0 && <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                <Typography color="textSecondary">This module does not create any resources</Typography>
            </Box>}
            {data.managedResources.length > 0 && <>
                <Typography color="textSecondary" variant="body1" mb={1}>
                    This module creates the following resources:
                </Typography>
                <List disablePadding>
                    {data.managedResources.map((resource) => (
                        <ListItem key={`${resource.type}.${resource.name}`} disableGutters>
                            <ListItemText
                                primary={<StyledCode children={`${resource.type}.${resource.name}`} />}
                            />
                        </ListItem>
                    ))}
                </List>
            </>}
        </Box>
    );
}

export default TerraformModuleVersionDocsResources;
