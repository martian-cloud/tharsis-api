import { useMemo } from 'react';
import Box from '@mui/material/Box';
import { TreeViewBaseItem } from '@mui/x-tree-view/models';
import { RichTreeView } from '@mui/x-tree-view/RichTreeView';
import { useFragment } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import { useSearchParams } from 'react-router-dom';
import { TerraformModuleVersionDocsSidebarFragment_configurationDetails$key } from './__generated__/TerraformModuleVersionDocsSidebarFragment_configurationDetails.graphql';

interface Props {
    onItemChange: (id: string) => void
    fragmentRef: TerraformModuleVersionDocsSidebarFragment_configurationDetails$key
}

function TerraformModuleVersionDocsSidebar({ fragmentRef, onItemChange }: Props) {
    const [searchParams] = useSearchParams();
    const item = searchParams.get('item');

    const data = useFragment<TerraformModuleVersionDocsSidebarFragment_configurationDetails$key>(
        graphql`
            fragment TerraformModuleVersionDocsSidebarFragment_configurationDetails on TerraformModuleConfigurationDetails {
                readme
                variables {
                    name
                }
                outputs {
                    name
                }
                managedResources {
                    name
                }
                dataResources {
                    name
                }
                requiredProviders {
                    source
                }
            }
        `, fragmentRef
    );

    const sidebarItems: TreeViewBaseItem[] = useMemo(() => {
        const items: TreeViewBaseItem[] = [];

        if (data.readme) {
            items.push({
                id: 'overview',
                label: 'Overview',
            });
        }

        items.push(
            {
                id: 'inputs',
                label: `Inputs (${data.variables.length || 0})`
            },
            {
                id: 'outputs',
                label: `Outputs (${data.outputs.length || 0})`,
            },
            {
                id: 'resources',
                label: `Resources (${data.managedResources.length || 0})`,
            },
            {
                id: 'dataSources',
                label: `Data Sources (${data.dataResources.length || 0})`,
            },
            {
                id: 'requiredProviders',
                label: `Required Providers (${data.requiredProviders.filter(provider => provider.source !== "").length})`,
            }
        );

        return items;
    }, [data]);

    const selectedItem = useMemo(() => {
        if (item) {
            return item;
        }

        if (data.readme) {
            return 'overview';
        }

        return 'inputs';
    }, [item, data.readme])

    return (
        <Box sx={{ height: "auto", minWidth: 250, mr: 1 }}>
            <RichTreeView
                onItemClick={(e, itemId: string) => onItemChange(itemId)}
                selectedItems={selectedItem}
                items={sidebarItems}
            />
        </Box>
    );
}

export default TerraformModuleVersionDocsSidebar;
