import Box from '@mui/material/Box';
import { TreeViewDefaultItemModelProperties } from '@mui/x-tree-view/models';
import { RichTreeView } from '@mui/x-tree-view/RichTreeView';
import graphql from 'babel-plugin-relay/macro';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import { TerraformProviderVersionDocsSidebarFragment_details$key } from './__generated__/TerraformProviderVersionDocsSidebarFragment_details.graphql';

interface Props {
    onItemChange: (id: string) => void
    fragmentRef: TerraformProviderVersionDocsSidebarFragment_details$key
}

interface Category {
    id: string
    label: string
}

const guides: Category = { id: 'guides', label: 'Guides' };
const functions: Category = { id: 'functions', label: 'Functions' };
const dataSources: Category = { id: 'data_sources', label: 'Data Sources' };
const ephemeralResources: Category = { id: 'ephemeral_resources', label: 'Ephemeral Resources' };
const resources: Category = { id: 'resources', label: 'Resources' };

// Sidebar id and default selection for the overview doc file.
export const OVERVIEW_ITEM_ID = 'overview/index.md';

// Categories that can be grouped under a subcategory, and the ones nested inside each subcategory folder.
const groupedCategories: Category[] = [dataSources, ephemeralResources, resources];

// Top-level section order (Overview is prepended and subcategory groups are appended separately).
const topLevelCategories: Category[] = [guides, dataSources, ephemeralResources, functions, resources];

interface DocItem {
    category: string
    filename: string
    // Subcategory group this item is shown under, or null when it is listed at the top level.
    group: string | null
}

// Folder ids are prefixed so they never contain "/"; the container treats a "/"-containing id as navigable.
const docFileId = (item: DocItem) => `${item.category}/${item.filename}`;
const groupId = (group: string) => `group:${encodeURIComponent(group)}`;
const sectionId = (group: string | null, categoryId: string) =>
    group === null ? `cat:${categoryId}` : `${groupId(group)}:${categoryId}`;

// docFileLabel is the filename shown in the sidebar, with the extension removed for a cleaner tree.
const docFileLabel = (filename: string) => filename.replace(/\.md$/, '');

// categorySection returns a section for the given category, or null when it has no files (empty categories are hidden).
function categorySection(group: string | null, category: Category, items: DocItem[]): TreeViewDefaultItemModelProperties | null {
    const files = items.filter(item => item.category === category.id);
    if (files.length === 0) {
        return null;
    }

    return {
        id: sectionId(group, category.id),
        label: `${category.label} (${files.length})`,
        children: files.map(file => ({ id: docFileId(file), label: docFileLabel(file.filename) })),
    };
}

// Folders that must be open for a doc file to be visible.
function ancestorIds(item: DocItem): string[] {
    const section = sectionId(item.group, item.category);
    return item.group === null ? [section] : [groupId(item.group), section];
}

function TerraformProviderVersionDocsSidebar({ fragmentRef, onItemChange }: Props) {
    const [searchParams] = useSearchParams();
    const selectedItem = searchParams.get('item') || OVERVIEW_ITEM_ID;

    const data = useFragment<TerraformProviderVersionDocsSidebarFragment_details$key>(
        graphql`
            fragment TerraformProviderVersionDocsSidebarFragment_details on TerraformProviderVersion {
                docs {
                    items {
                        filename
                        category
                        subcategory
                    }
                }
            }
        `, fragmentRef
    );

    // ids are lower_snake_case; the GraphQL category enum is UPPER_SNAKE_CASE.
    const docItems: DocItem[] = useMemo(
        () => data.docs.items.map(item => {
            const category = item.category.toLowerCase();
            const isGroupable = groupedCategories.some(c => c.id === category);
            return {
                category,
                filename: item.filename,
                group: isGroupable && item.subcategory ? item.subcategory : null,
            };
        }),
        [data.docs.items]
    );

    const sidebarItems: TreeViewDefaultItemModelProperties[] = useMemo(() => {
        const items: TreeViewDefaultItemModelProperties[] = [{ id: OVERVIEW_ITEM_ID, label: 'Overview' }];

        const groups = [...new Set(docItems.flatMap(item => item.group === null ? [] : [item.group]))].sort();
        const ungrouped = docItems.filter(item => item.group === null);

        // Top-level sections (empty ones are hidden): Guides, Data Sources, Ephemeral Resources, Functions, Resources.
        for (const category of topLevelCategories) {
            const section = categorySection(null, category, ungrouped);
            if (section) {
                items.push(section);
            }
        }
        for (const group of groups) {
            const groupItems = docItems.filter(item => item.group === group);
            const children = groupedCategories
                .map(category => categorySection(group, category, groupItems))
                .filter((section): section is TreeViewDefaultItemModelProperties => section !== null);

            items.push({ id: groupId(group), label: group, children });
        }

        return items;
    }, [docItems]);

    const ancestorsOf = useCallback((itemId: string): string[] => {
        const item = docItems.find(docItem => docFileId(docItem) === itemId);
        return item ? ancestorIds(item) : [];
    }, [docItems]);

    const [expandedItems, setExpandedItems] = useState<string[]>(() => ancestorsOf(selectedItem));

    // Open the selected doc file's folders on mount and whenever the URL changes (deep link, back/forward).
    useEffect(() => {
        setExpandedItems(prev => [...new Set([...prev, ...ancestorsOf(selectedItem)])]);
    }, [selectedItem, ancestorsOf]);

    return (
        <Box
            sx={{
                width: 320,
                flexShrink: 0,
                mr: 1,
                maxHeight: '70vh',
                overflowY: 'auto',
            }}
        >
            <RichTreeView
                onItemClick={(_e, itemId: string) => onItemChange(itemId)}
                selectedItems={selectedItem}
                expandedItems={expandedItems}
                onExpandedItemsChange={(_e, itemIds: string[]) => setExpandedItems(itemIds)}
                items={sidebarItems}
                sx={{
                    // Wrap long filenames onto multiple lines instead of truncating them with an ellipsis.
                    '& .MuiTreeItem-label': {
                        wordBreak: 'break-word',
                    },
                }}
            />
        </Box>
    );
}

export default TerraformProviderVersionDocsSidebar;
