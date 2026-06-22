import { TreeViewDefaultItemModelProperties } from '@mui/x-tree-view/models';
import { ArchiveFile } from './tarball';

// buildFileTree nests file paths; folders sort before files at each level.
export function buildFileTree(files: ArchiveFile[]): TreeViewDefaultItemModelProperties[] {
    const root: TreeViewDefaultItemModelProperties[] = [];

    for (const file of files) {
        const segments = file.path.split('/');
        let level = root;
        let prefix = '';

        segments.forEach((segment, index) => {
            prefix = prefix ? `${prefix}/${segment}` : segment;

            if (index === segments.length - 1) {
                level.push({ id: file.path, label: segment });
                return;
            }

            // trailing slash keeps folder ids distinct from a same-named file
            let folder = level.find((item) => item.label === segment && !!item.children);
            if (!folder) {
                folder = { id: `${prefix}/`, label: segment, children: [] };
                level.push(folder);
            }

            level = folder.children!;
        });
    }

    sortLevel(root);

    return root;
}

// collectFolderIds returns the ids of every folder node so the tree can be fully expanded.
export function collectFolderIds(items: TreeViewDefaultItemModelProperties[]): string[] {
    const ids: string[] = [];

    for (const item of items) {
        if (item.children) {
            ids.push(item.id);
            ids.push(...collectFolderIds(item.children));
        }
    }

    return ids;
}

// pickDefaultFile prefers the named root file, else the first root file, else the first file.
export function pickDefaultFile(files: ArchiveFile[], preferred?: string): string | undefined {
    if (files.length === 0) {
        return undefined;
    }

    const rootFiles = files.filter((file) => !file.path.includes('/'));
    if (preferred) {
        const match = rootFiles.find((file) => file.path === preferred);
        if (match) {
            return match.path;
        }
    }

    return (rootFiles[0] || files[0]).path;
}

// sortLevel orders a tree level with folders first, then files, alphabetically within each group.
function sortLevel(items: TreeViewDefaultItemModelProperties[]) {
    items.sort((a, b) => {
        const aFolder = !!a.children;
        const bFolder = !!b.children;
        if (aFolder !== bFolder) {
            return aFolder ? -1 : 1;
        }

        return a.label.localeCompare(b.label);
    });

    for (const item of items) {
        if (item.children) {
            sortLevel(item.children);
        }
    }
}
