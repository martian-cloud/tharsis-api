import { useState, useCallback } from 'react';
import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Typography,
    Divider
} from '@mui/material';
import { Edit as EditIcon } from '@mui/icons-material';
import LabelEditor from './LabelEditor';
import { Label } from './types';
import LabelList from './LabelList';
import { sanitizeLabels } from './labelErrorHandling';
import { DEFAULT_MAX_LABELS } from './types';

interface Props {
    labels: Label[];
    onSave: (labels: Label[]) => Promise<void>;
    disabled?: boolean;
    maxLabels?: number;
    title?: string;
    description?: string;
}

function LabelManager({
    labels,
    onSave,
    disabled = false,
    maxLabels = DEFAULT_MAX_LABELS,
    title = "Manage Labels",
    description = "Add, edit, or remove labels to organize and categorize this workspace."
}: Props) {
    const [isOpen, setIsOpen] = useState(false);
    const [editingLabels, setEditingLabels] = useState<Label[]>([]);
    const [isSaving, setIsSaving] = useState(false);
    const [error, setError] = useState<string>('');
    const [showValidationErrors, setShowValidationErrors] = useState(false);

    const handleOpen = useCallback(() => {
        setEditingLabels([...labels]);
        setError('');
        setShowValidationErrors(false);
        setIsOpen(true);
    }, [labels]);

    const handleClose = useCallback(() => {
        setIsOpen(false);
        setEditingLabels([]);
        setError('');
        setShowValidationErrors(false);
    }, []);

    const handleSave = useCallback(async () => {
        try {
            setIsSaving(true);
            setError('');
            setShowValidationErrors(true);

            // Check for incomplete labels (key without value or value without key)
            const incompleteLabels = editingLabels.filter(label => {
                const hasKey = label.key.trim().length > 0;
                const hasValue = label.value.trim().length > 0;
                return (hasKey && !hasValue) || (!hasKey && hasValue);
            });

            if (incompleteLabels.length > 0) {
                setError('All labels must have both a key and a value. Please complete or remove incomplete labels.');
                setIsSaving(false);
                return;
            }

            // Check for duplicate keys
            const labelKeys = editingLabels
                .filter(label => label.key.trim().length > 0)
                .map(label => label.key.trim());

            const duplicateKeys = labelKeys.filter((key, index) => labelKeys.indexOf(key) !== index);

            if (duplicateKeys.length > 0) {
                const uniqueDuplicates = [...new Set(duplicateKeys)];
                setError(`Duplicate label keys found: ${uniqueDuplicates.join(', ')}. Each label key must be unique.`);
                setIsSaving(false);
                return;
            }

            // Sanitize labels before saving (remove empty ones, trim whitespace)
            const sanitizedLabels = sanitizeLabels(editingLabels);

            await onSave(sanitizedLabels);
            handleClose();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'An unexpected error occurred');
        } finally {
            setIsSaving(false);
        }
    }, [editingLabels, onSave, handleClose]);

    const hasChanges = () => {
        if (labels.length !== editingLabels.length) return true;

        return labels.some((label, index) => {
            const editingLabel = editingLabels[index];
            return !editingLabel ||
                   label.key !== editingLabel.key ||
                   label.value !== editingLabel.value;
        });
    };

    const getValidationErrors = (): Array<{ index: number; field: 'key' | 'value'; message: string }> => {
        const errors: Array<{ index: number; field: 'key' | 'value'; message: string }> = [];

        // Track seen keys for duplicate detection
        const seenKeys = new Map<string, number>();

        editingLabels.forEach((label, index) => {
            const hasKey = label.key.trim().length > 0;
            const hasValue = label.value.trim().length > 0;

            // Check for incomplete labels
            if (hasKey && !hasValue) {
                errors.push({ index, field: 'value', message: 'Value is required' });
            } else if (!hasKey && hasValue) {
                errors.push({ index, field: 'key', message: 'Key is required' });
            }

            // Check for duplicate keys (only for non-empty keys)
            if (hasKey) {
                const trimmedKey = label.key.trim();
                const originalIndex = seenKeys.get(trimmedKey);
                if (originalIndex !== undefined) {
                    // Mark both the original and duplicate
                    errors.push({ index: originalIndex, field: 'key', message: 'Duplicate key' });
                    errors.push({ index, field: 'key', message: 'Duplicate key' });
                } else {
                    seenKeys.set(trimmedKey, index);
                }
            }
        });

        return errors;
    };

    return (
        <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <Typography variant="subtitle1">Labels</Typography>
                <Button
                    size="small"
                    startIcon={<EditIcon />}
                    onClick={handleOpen}
                    disabled={disabled}
                    variant="outlined"
                >
                    Manage Labels
                </Button>
            </Box>

            <Box sx={{ mb: 2 }}>
                <LabelList
                    labels={labels}
                    showEmptyState={true}
                    emptyStateText="No labels assigned"
                    maxVisible={10}
                />
            </Box>

            <Dialog
                open={isOpen}
                onClose={handleClose}
                maxWidth="md"
                fullWidth
                PaperProps={{
                    sx: { minHeight: '400px' }
                }}
            >
                <DialogTitle>
                    <Typography component="div" variant="h6">{title}</Typography>
                    <Typography variant="body2" color="textSecondary" sx={{ mt: 1 }}>
                        {description}
                    </Typography>
                </DialogTitle>

                <Divider />

                <DialogContent sx={{ pt: 3 }}>
                    <LabelEditor
                        labels={editingLabels}
                        onChange={setEditingLabels}
                        error={error}
                        maxLabels={maxLabels}
                        disabled={isSaving}
                        validationErrors={showValidationErrors ? getValidationErrors() : []}
                    />
                </DialogContent>

                <Divider />

                <DialogActions sx={{ p: 2 }}>
                    <Button
                        onClick={handleClose}
                        disabled={isSaving}
                    >
                        Cancel
                    </Button>
                    <Button
                        onClick={handleSave}
                        disabled={!hasChanges() || isSaving}
                        variant="contained"
                    >
                        {isSaving ? 'Saving...' : 'Save Changes'}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}

export default LabelManager;
