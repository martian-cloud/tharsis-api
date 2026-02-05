import {
    Add as AddIcon,
    Clear as ClearIcon,
    FilterList as FilterListIcon
} from '@mui/icons-material';
import {
    Box,
    Button,
    Chip,
    Collapse,
    Divider,
    Paper,
    Stack,
    TextField,
    Typography
} from '@mui/material';
import { useTheme } from '@mui/material/styles';
import { useCallback, useState } from 'react';


export interface LabelFilterItem {
    key: string;
    value: string;
}

interface Props {
    filters: LabelFilterItem[];
    onFiltersChange: (filters: LabelFilterItem[]) => void;
    expanded?: boolean;
}

function LabelFilter({
    filters,
    onFiltersChange,
    expanded = false,
}: Props) {
    const theme = useTheme();
    const [newFilter, setNewFilter] = useState<Partial<LabelFilterItem>>({});

    const handleAddFilter = useCallback(() => {
        if (newFilter.key && newFilter.value) {
            // Check if this exact filter key already exists
            const exists = filters.some(f => f.key === newFilter.key);
            if (!exists) {
                onFiltersChange([...filters, newFilter as LabelFilterItem]);
                setNewFilter({});
            }
        }
    }, [filters, newFilter, onFiltersChange]);

    const handleRemoveFilter = useCallback((index: number) => {
        const newFilters = filters.filter((_, i) => i !== index);
        onFiltersChange(newFilters);
    }, [filters, onFiltersChange]);

    const handleClearAll = useCallback(() => {
        onFiltersChange([]);
        setNewFilter({});
    }, [onFiltersChange]);

    const canAddFilter = newFilter.key && newFilter.value && !filters.some(f => f.key === newFilter.key);
    const hasFilters = filters.length > 0;

    return (
        <Paper
            variant="outlined"
            sx={{
                mb: 2,
                overflow: 'hidden',
                borderColor: hasFilters ? theme.palette.primary.main : theme.palette.divider,
                borderWidth: hasFilters ? 2 : 1,
            }}
        >
            {/* Header */}
            <Box
                sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    p: 2
                }}
            >
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <FilterListIcon color={hasFilters ? 'primary' : 'action'} />
                    <Typography variant="subtitle2" color={hasFilters ? 'primary' : 'textPrimary'}>
                        Filter by Labels
                    </Typography>
                    {hasFilters && (
                        <Chip
                            size="small"
                            label={filters.length}
                            color="primary"
                            variant="filled"
                        />
                    )}
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    {hasFilters && (
                        <Button
                            size="small"
                            onClick={(e) => {
                                e.stopPropagation();
                                handleClearAll();
                            }}
                            startIcon={<ClearIcon />}
                            color="primary"
                        >
                            Clear All
                        </Button>
                    )}
                </Box>
            </Box>

            {/* Collapsible Content */}
            <Collapse in={expanded}>
                <Divider />
                <Box sx={{ p: 2 }}>
                    {/* Active Filters */}
                    {hasFilters && (
                        <Box sx={{ mb: 2 }}>
                            <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                                Active Filters (AND logic):
                            </Typography>
                            <Stack direction="row" spacing={1} sx={{ flexWrap: 'wrap', gap: 1 }}>
                                {filters.map((filter, index) => (
                                    <Chip
                                        key={`${filter.key}-${filter.value}-${index}`}
                                        label={`${filter.key}: ${filter.value}`}
                                        onDelete={() => handleRemoveFilter(index)}
                                        color="primary"
                                        variant="filled"
                                        size="small"
                                    />
                                ))}
                            </Stack>
                        </Box>
                    )}

                    {/* Add New Filter */}
                    <Box>
                        <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
                            Add Label Filter:
                        </Typography>
                        <Stack direction="row" spacing={2} alignItems="flex-start">
                            <Box sx={{ minWidth: 200 }}>
                                <TextField
                                    size="small"
                                    label="Label Key"
                                    placeholder="e.g., environment"
                                    value={newFilter.key || ''}
                                    onChange={(e) => setNewFilter(prev => ({ ...prev, key: e.target.value, value: '' }))}
                                    sx={{ width: '100%' }}
                                />
                            </Box>
                            <Box sx={{ minWidth: 200 }}>
                                <TextField
                                    size="small"
                                    label="Label Value"
                                    placeholder="e.g., production"
                                    value={newFilter.value || ''}
                                    onChange={(e) => setNewFilter(prev => ({ ...prev, value: e.target.value }))}
                                    disabled={!newFilter.key}
                                    sx={{ width: '100%' }}
                                />
                            </Box>
                            <Button
                                variant="contained"
                                startIcon={<AddIcon />}
                                onClick={handleAddFilter}
                                disabled={!canAddFilter}
                                sx={{ mt: 0.5 }}
                            >
                                Add
                            </Button>
                        </Stack>
                    </Box>
                </Box>
            </Collapse>
        </Paper>
    );
}

export default LabelFilter;
