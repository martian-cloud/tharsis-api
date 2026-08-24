import Editor, { OnMount } from '@monaco-editor/react';
// Side-effect import: configures Monaco to load locally (bundled) rather than
// from the CDN, which our CSP (script-src 'self') blocks. Must run before <Editor>.
import '../../common/monaco';
import { Box, Button, IconButton, List, ListItemButton, ListItemText, Paper, Stack, TextField, Typography } from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import { useState } from 'react';
import { PolicyFile, newPolicyFile } from './policyFiles';

interface Props {
    files: PolicyFile[];
    onChange: (files: PolicyFile[]) => void;
}

// PackageFilesEditor is the shared file-list + Monaco editor used by both the create-version and
// edit-version pages: a left-hand file list (add/remove/select) and a right-hand filename field
// plus a Rego-highlighted Monaco editor for the selected file.
function PackageFilesEditor({ files, onChange }: Props) {
    // Selection is tracked by id rather than by position: an index follows whichever file happens to
    // sit in that slot, so removing a file above the selected one would silently move the selection
    // onto a different file. Null until the user picks one, which falls back to the first file below.
    const [selectedId, setSelectedId] = useState<string | null>(null);

    // Undefined only when there are no files at all. An unknown id -- nothing picked yet, or the
    // selected file was removed -- falls back to the first file.
    const selectedFile = files.find(file => file._id === selectedId) ?? files[0];

    const onAddFile = () => {
        const added = newPolicyFile('', '');
        setSelectedId(added._id);
        onChange([...files, added]);
    };

    const onRemoveFile = (id: string) => {
        const index = files.findIndex(file => file._id === id);
        const next = files.filter(file => file._id !== id);
        // Removing the selected file lands on whatever takes its place -- the file below it, or the
        // new last file when the bottom one went -- rather than jumping back to the top of the list.
        if (id === selectedFile?._id) {
            setSelectedId(next[Math.min(index, next.length - 1)]?._id ?? null);
        }
        onChange(next);
    };

    const onFilenameChange = (id: string, filename: string) => {
        onChange(files.map(file => (file._id === id ? { ...file, name: filename } : file)));
    };

    const onContentChange = (id: string, content: string) => {
        onChange(files.map(file => (file._id === id ? { ...file, content } : file)));
    };

    // Every file gets its own Monaco model (see the path prop below), so EOL normalisation has to run
    // on each model swap and not just at mount -- onMount fires once for the editor, not once per
    // file. A tarball authored on Windows arrives with CRLF and is re-tarred verbatim on save, so
    // without this only the first file opened would be normalised.
    const onEditorMount: OnMount = (editor, monaco) => {
        const normalizeEOL = () => editor.getModel()?.pushEOL(monaco.editor.EndOfLineSequence.LF);
        normalizeEOL();
        // Disposed along with the editor, so there is nothing to unsubscribe here.
        editor.onDidChangeModel(normalizeEOL);
    };

    return (
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ height: '100%', minHeight: 0 }}>
            <Paper
                variant="outlined"
                sx={{
                    width: { xs: '100%', md: 280 },
                    flexShrink: 0,
                    display: 'flex',
                    flexDirection: 'column',
                    minHeight: 0,
                    maxHeight: { xs: 240, md: 'none' },
                }}
            >
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: 1 }}>
                    <Typography variant="subtitle2">Files</Typography>
                    <Button size="small" variant="outlined" color="secondary" onClick={onAddFile}>Add file</Button>
                </Box>
                <List dense disablePadding sx={{ flexGrow: 1, minHeight: 0, overflowY: 'auto' }}>
                    {files.map(file => (
                        <ListItemButton
                            key={file._id}
                            selected={file._id === selectedFile?._id}
                            onClick={() => setSelectedId(file._id)}
                        >
                            <ListItemText
                                primary={file.name || '(unnamed)'}
                                primaryTypographyProps={{ noWrap: true }}
                            />
                            <IconButton
                                aria-label='Remove file'
                                edge="end"
                                size="small"
                                onClick={event => { event.stopPropagation(); onRemoveFile(file._id); }}
                            >
                                <DeleteIcon fontSize="small" />
                            </IconButton>
                        </ListItemButton>
                    ))}
                </List>
            </Paper>

            <Box sx={{ flexGrow: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
                {selectedFile && <>
                    <TextField
                        label="Filename"
                        size="small"
                        fullWidth
                        margin="none"
                        sx={{ marginBottom: 1, flexShrink: 0 }}
                        value={selectedFile.name}
                        placeholder="policy.rego"
                        onChange={event => onFilenameChange(selectedFile._id, event.target.value)}
                    />
                    <Box sx={{ flexGrow: 1, minHeight: 0, border: 1, borderColor: 'divider' }}>
                        <Editor
                            theme="vs-dark"
                            defaultLanguage="rego"
                            value={selectedFile.content}
                            // Keying the model by the row's id gives each file its own undo stack and
                            // its own saved cursor/scroll position, instead of one shared model whose
                            // history bleeds across files. It has to be the id and not the filename:
                            // the filename changes on every keystroke in the field above, and repeats
                            // across mounts, which would hand back a stale cached model.
                            path={`inmemory://policy-files/${selectedFile._id}`}
                            onChange={value => onContentChange(selectedFile._id, value ?? '')}
                            onMount={onEditorMount}
                            options={{
                                fontSize: 12,
                                minimap: { enabled: false },
                                scrollBeyondLastLine: false,
                                automaticLayout: true,
                                wordWrap: 'on',
                            }}
                        />
                    </Box>
                </>}
            </Box>
        </Stack>
    );
}

export default PackageFilesEditor;
