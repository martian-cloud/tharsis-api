import { useEffect } from 'react';
import {
    Box,
    IconButton,
    TextField,
    Typography,
    Alert,
    Stack,
    Chip
} from '@mui/material';
import { Delete as DeleteIcon } from '@mui/icons-material';

import { sanitizeLabels } from './labelErrorHandling';
import { DEFAULT_MAX_LABELS, Label } from './types';

export interface LabelValidationError {
    index: number;
    field: 'key' | 'value';
    message: string;
}

interface Props {
    labels: Label[];
    onChange: (labels: Label[]) => void;
    error?: string;
    maxLabels?: number;
    disabled?: boolean;
    validationErrors?: LabelValidationError[];
}

function LabelEditor({
    labels,
    onChange,
    error,
    maxLabels = DEFAULT_MAX_LABELS,
    disabled = false,
    validationErrors = []
}: Props) {
    const getFieldError = (index: number, field: 'key' | 'value'): string | undefined => {
        return validationErrors.find(e => e.index === index && e.field === field)?.message;
    };
    // Ensure there's always at least one empty label for better UX
    useEffect(() => {
        if (labels.length === 0) {
            onChange([{ key: '', value: '' }]);
        } else {
            // Check if there's already an empty row available for adding new labels
            const hasEmptyRow = labels.some(label => !label.key.trim() && !label.value.trim());
            if (!hasEmptyRow && labels.length < maxLabels) {
                onChange([...labels, { key: '', value: '' }]);
            }
        }
    }, [labels, maxLabels, onChange]);

    const handleLabelChange = (index: number, field: 'key' | 'value', newValue: string) => {
        const updatedLabels = [...labels];
        updatedLabels[index] = { ...updatedLabels[index], [field]: newValue };

        // Auto-add new empty label if user is typing in the last label and it's not empty
        const isLastLabel = index === labels.length - 1;
        const hasContent = newValue.trim().length > 0;
        if (isLastLabel && hasContent && labels.length < maxLabels) {
            updatedLabels.push({ key: '', value: '' });
        }

        onChange(updatedLabels);
    };

    const handleRemoveLabel = (index: number) => {
        const newLabels = labels.filter((_, i) => i !== index);
        onChange(newLabels);
    };

    return (
        <Box>
            <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="textPrimary">
                    Labels ({sanitizeLabels(labels).length}/{maxLabels})
                </Typography>
                <Typography variant="caption" color="textSecondary">
                    Start typing to add labels. New rows are added automatically.
                </Typography>
            </Box>

            {error && (
                <Alert severity="error" sx={{ mb: 2 }}>
                    {error}
                </Alert>
            )}

            <Stack spacing={2}>
                {labels.map((label, index) => (
                    <Box key={index} sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
                        <Box sx={{ flex: 1 }}>
                            <TextField
                                size="small"
                                label="Key"
                                placeholder="Enter label key..."
                                value={label.key}
                                onChange={(e) => handleLabelChange(index, 'key', e.target.value)}
                                error={!!getFieldError(index, 'key')}
                                helperText={getFieldError(index, 'key')}
                                disabled={disabled}
                                sx={{ width: '100%' }}
                                inputProps={{ maxLength: 63 }}
                            />
                        </Box>
                        <Box sx={{ flex: 2 }}>
                            <TextField
                                size="small"
                                label="Value"
                                placeholder="Enter label value..."
                                value={label.value}
                                onChange={(e) => handleLabelChange(index, 'value', e.target.value)}
                                error={!!getFieldError(index, 'value')}
                                helperText={getFieldError(index, 'value')}
                                disabled={disabled}
                                sx={{ width: '100%' }}
                                inputProps={{ maxLength: 255 }}
                            />
                        </Box>
                        <IconButton
                            size="small"
                            onClick={() => handleRemoveLabel(index)}
                            disabled={disabled || (labels.length === 1 && !label.key && !label.value)}
                            color="error"
                            sx={{ mt: 0.5 }}
                            aria-label={`Delete label ${label.key || 'empty'}`}
                            title="Delete label"
                        >
                            <DeleteIcon />
                        </IconButton>
                    </Box>
                ))}
            </Stack>

            {labels.length > 0 && (
                <Box sx={{ mt: 2 }}>
                    <Typography variant="caption" color="textSecondary">
                        Preview:
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1 }}>
                        {labels
                            .filter(label => label.key && label.value)
                            .map((label, index) => (
                                <Chip
                                    key={index}
                                    size="small"
                                    variant="outlined"
                                    label={`${label.key}: ${label.value}`}
                                />
                            ))}
                    </Box>
                </Box>
            )}
        </Box>
    );
}

export default LabelEditor;
