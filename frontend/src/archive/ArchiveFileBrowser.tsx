import DownloadIcon from '@mui/icons-material/Download';
import { Alert, Box, Button, CircularProgress, Typography, useTheme } from '@mui/material';
import { RichTreeView } from '@mui/x-tree-view/RichTreeView';
import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import CopyButton from '../common/CopyButton';
import downloadFile from '../common/filedownload';
import { buildFileTree, collectFolderIds, pickDefaultFile } from './filetree';
import { ArchiveFile, ArchiveTooLargeError, decodeText, extractTarGz, languageForFile } from './tarball';

interface Props {
    load: () => Promise<ArrayBuffer>;
    // downloadName maps a file path to the saved filename; defaults to the path's basename.
    downloadName?: (path: string) => string;
    preferredFile?: string;
}

function ArchiveFileBrowser({ load, downloadName, preferredFile }: Props) {
    const theme = useTheme();
    const [searchParams, setSearchParams] = useSearchParams();

    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<'generic' | 'tooLarge' | null>(null);
    const [files, setFiles] = useState<ArchiveFile[]>([]);

    useEffect(() => {
        let active = true;

        setLoading(true);
        setError(null);

        load()
            .then((data) => extractTarGz(data))
            .then((extracted) => {
                if (active) {
                    setFiles(extracted);
                }
            })
            .catch((err) => {
                if (active) {
                    console.error('failed to load archive files', err);
                    setError(err instanceof ArchiveTooLargeError ? 'tooLarge' : 'generic');
                }
            })
            .finally(() => {
                if (active) {
                    setLoading(false);
                }
            });

        return () => {
            active = false;
        };
    }, [load]);

    const tree = useMemo(() => buildFileTree(files), [files]);
    const expandedItems = useMemo(() => collectFolderIds(tree), [tree]);
    const filePaths = useMemo(() => new Set(files.map((file) => file.path)), [files]);

    // Selected file and highlighted line live in the URL so a view can be shared.
    const fileParam = searchParams.get('file');
    const selected = (fileParam && filePaths.has(fileParam)) ? fileParam : pickDefaultFile(files, preferredFile);
    const selectedFile = files.find((file) => file.path === selected);

    const lineParam = Number(searchParams.get('line'));
    const activeLine = Number.isInteger(lineParam) && lineParam > 0 ? lineParam : undefined;

    const onSelectFile = (path: string) => {
        const next = new URLSearchParams(searchParams);
        next.set('file', path);
        next.delete('line');
        setSearchParams(next, { replace: true, state: { preventScrollReset: true } });
    };

    const onSelectLine = useCallback((line: number) => {
        const next = new URLSearchParams(searchParams);
        if (selected) {
            next.set('file', selected);
        }
        next.set('line', String(line));
        setSearchParams(next, { replace: true, state: { preventScrollReset: true } });
    }, [searchParams, selected, setSearchParams]);

    const onDownloadFile = useCallback((file: ArchiveFile) => {
        const name = downloadName ? downloadName(file.path) : (file.path.split('/').pop() ?? file.path);
        downloadFile(name, new Blob([new Uint8Array(file.data)]));
    }, [downloadName]);

    if (loading) {
        return (
            <Box display="flex" justifyContent="center" alignItems="center" padding={4}>
                <CircularProgress />
            </Box>
        );
    }

    if (error === 'tooLarge') {
        return <Alert severity="info">This archive is too large to preview. Download it to view the contents.</Alert>;
    }

    if (error) {
        return <Alert severity="error">Unable to load the files. Please try again or download the archive instead.</Alert>;
    }

    if (files.length === 0) {
        return (
            <Box padding={2} display="flex" justifyContent="center" alignItems="center">
                <Typography color="textSecondary">No files in this archive</Typography>
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
                    '& > *': { mb: 4 },
                }
            }}
        >
            <Box sx={{ minWidth: 250, mr: 1 }}>
                <RichTreeView
                    items={tree}
                    defaultExpandedItems={expandedItems}
                    selectedItems={selected ?? null}
                    onItemClick={(_event, itemId: string) => {
                        if (filePaths.has(itemId)) {
                            onSelectFile(itemId);
                        }
                    }}
                />
            </Box>
            <Box
                flex={1}
                pl={2}
                borderLeft={`1px solid ${theme.palette.divider}`}
                sx={{
                    overflowX: 'auto',
                    [theme.breakpoints.down('lg')]: {
                        borderLeft: 'none',
                        paddingLeft: 0,
                        minWidth: 0,
                    }
                }}
            >
                {selectedFile && <FileContent
                    file={selectedFile}
                    activeLine={activeLine}
                    onSelectLine={onSelectLine}
                    onDownload={onDownloadFile}
                />}
            </Box>
        </Box>
    );
}

interface FileContentProps {
    file: ArchiveFile;
    activeLine?: number;
    onSelectLine: (line: number) => void;
    onDownload: (file: ArchiveFile) => void;
}

const FileContent = memo(function FileContent({ file, activeLine, onSelectLine, onDownload }: FileContentProps) {
    const decoded = useMemo(() => decodeText(file.data), [file]);

    const activeLineRef = useRef(activeLine);
    activeLineRef.current = activeLine;

    useEffect(() => {
        // Scroll to a deep-linked line only when a file first opens, not on subsequent
        // line clicks, which would otherwise jump the page away from the user's click.
        const line = activeLineRef.current;
        if (line) {
            document.getElementById(`L${line}`)?.scrollIntoView({ block: 'center' });
        }
    }, [file]);

    if (decoded.binary) {
        return (
            <Box padding={2} display="flex" flexDirection="column" alignItems="center" gap={2}>
                <Typography color="textSecondary">This file can't be previewed.</Typography>
                <Button size="small" color="info" variant="outlined" startIcon={<DownloadIcon />} onClick={() => onDownload(file)}>
                    Download
                </Button>
            </Box>
        );
    }

    return (
        <Box>
            {decoded.truncated && <Alert severity="warning" sx={{ marginBottom: 2 }}>
                This file is too large to preview in full. Showing a truncated preview — download to view the full contents.
            </Alert>}
            <Box position="relative">
                <Box sx={{ position: 'absolute', top: 4, right: 4, zIndex: 1 }}>
                    <CopyButton data={decoded.fullText} toolTip="Copy file contents" />
                </Box>
                <SyntaxHighlighter
                    showLineNumbers
                    wrapLines
                    lineProps={(lineNumber: number) => ({
                        id: `L${lineNumber}`,
                        // ignore clicks that finish a text selection
                        onClick: () => window.getSelection()?.isCollapsed !== false && onSelectLine(lineNumber),
                        style: {
                            display: 'block',
                            cursor: 'pointer',
                            backgroundColor: lineNumber === activeLine ? 'rgba(255, 255, 255, 0.1)' : undefined,
                        },
                    })}
                    customStyle={{ fontSize: 14, margin: 0 }}
                    language={languageForFile(file.path)}
                    style={prismTheme}
                >
                    {decoded.text}
                </SyntaxHighlighter>
            </Box>
        </Box>
    );
});

export default ArchiveFileBrowser;
