import Editor from '@monaco-editor/react';
// Side-effect import: configures Monaco to load locally (bundled) rather than
// from the CDN, which our CSP (script-src 'self') blocks. Must run before <Editor>.
import '../../common/monaco';
import { Box, MenuItem, Paper, Stack, TextField } from '@mui/material';
import { useState } from 'react';
import { SAMPLE_OPA_INPUTS } from './opaSamples';

// OPASampleInputsPanel is the read-only reference panel shown alongside the policy editor: an example
// of the input document a policy is evaluated against, one per stage. It is shared by the create- and
// edit-version pages so both show the same examples and only one copy has to track changes to the
// input document's shape.
//
// The selected example is local state: the panel is the only thing that cares which one is showing,
// and it is remounted (losing the selection) only when the page hides it entirely.
function OPASampleInputsPanel() {
    const [sampleInputId, setSampleInputId] = useState(SAMPLE_OPA_INPUTS[0].id);

    const selectedSample = SAMPLE_OPA_INPUTS.find(sample => sample.id === sampleInputId) ?? SAMPLE_OPA_INPUTS[0];

    return (
        <Paper
            variant="outlined"
            sx={{ width: 400, flexShrink: 0, display: 'flex', flexDirection: 'column' }}
        >
            <Stack
                direction="row"
                alignItems="center"
                justifyContent="space-between"
                gap={1}
                sx={{ px: 2, py: 1, borderBottom: 1, borderColor: 'divider', bgcolor: 'action.hover', minHeight: 48 }}
            >
                <TextField
                    select
                    size="small"
                    label="OPA Input Example"
                    value={sampleInputId}
                    onChange={event => setSampleInputId(event.target.value)}
                    fullWidth
                >
                    {SAMPLE_OPA_INPUTS.map(sample => (
                        <MenuItem key={sample.id} value={sample.id}>{sample.label}</MenuItem>
                    ))}
                </TextField>
            </Stack>
            <Box sx={{ flexGrow: 1, minHeight: 0 }}>
                <Editor
                    height="100%"
                    theme="vs-dark"
                    defaultLanguage="json"
                    path={`sample-${selectedSample.id}.json`}
                    value={selectedSample.content}
                    onMount={(editor, monaco) => editor.getModel()?.pushEOL(monaco.editor.EndOfLineSequence.LF)}
                    options={{
                        readOnly: true,
                        fontSize: 12,
                        minimap: { enabled: false },
                        scrollBeyondLastLine: false,
                        automaticLayout: true,
                        wordWrap: 'on',
                    }}
                />
            </Box>
        </Paper>
    );
}

export default OPASampleInputsPanel;
