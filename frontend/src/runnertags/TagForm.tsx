import { Button, Chip, Paper, Stack, TextField } from '@mui/material';
import { useState } from 'react';

interface Props {
    data: readonly string[]
    onChange: (data: { tags: string[] }) => void
}

function TagForm({ data, onChange }: Props) {
    const [tagToAdd, setTagToAdd] = useState('');

    const onAddTag = () => {
        onChange({ tags: [...data, tagToAdd] });
        setTagToAdd('');
    };

    const onRemoveTag = (tag: string) => {
        const index = data.indexOf(tag);
        if (index !== -1) {
            const tagsCopy = [...data];
            tagsCopy.splice(index, 1);
            onChange({ ...data, tags: tagsCopy });
        }
    };

    return (
        <Paper sx={{ padding: 2 }} variant="outlined">
            <Paper sx={{ padding: 2, display: 'flex', alignItems: 'center', mb: data.length > 0 ? 2 : 0 }}>
                <TextField
                    size="small"
                    margin="none"
                    sx={{ flex: 1, mr: 1 }}
                    fullWidth
                    value={tagToAdd}
                    placeholder="Enter tag to add"
                    variant="standard"
                    color="secondary"
                    onChange={event => setTagToAdd(event.target.value)}
                />
                <Button
                    onClick={onAddTag}
                    disabled={tagToAdd === ''}
                    variant="outlined"
                    color="secondary">
                    Add Tag
                </Button>
            </Paper>
            <Stack direction="row" spacing={2}>
                {data.map(tag => <Chip key={tag} color="secondary" label={tag} onDelete={() => onRemoveTag(tag)} />)}
            </Stack>
        </Paper>
    );
}

export default TagForm;
