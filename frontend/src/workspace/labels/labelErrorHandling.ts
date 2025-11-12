// Error handling utilities for label management
import { Label } from './types';

/**
 * Sanitize labels (remove empty ones, trim whitespace)
 * Use this for both validation/counting and before saving
 */
export const sanitizeLabels = (labels: Label[]): Label[] => {
    return labels
        .filter(label => label.key.trim() && label.value.trim())
        .map(label => ({
            key: label.key.trim(),
            value: label.value.trim()
        }));
};
