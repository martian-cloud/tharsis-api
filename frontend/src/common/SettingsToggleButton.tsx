import { Box, Button, Typography } from '@mui/material';
import { useAppHeaderHeight } from '../contexts/AppHeaderHeightProvider';

// settingsSectionElementId is the id of a section's header, which a section requested on load is scrolled to.
export function settingsSectionElementId(section: string) {
    return `settings-section-${section}`;
}

interface Props {
    title: string;
    // section is the key used to expand this section from the `section` query param
    section?: string;
    showSettings: boolean;
    onToggle: () => void;
}

function SettingsToggleButton({ title, section, showSettings, onToggle }: Props) {
    const { headerHeight } = useAppHeaderHeight();

    return (
        // A section scrolled to on load stops 16px below the fixed app header rather than underneath it
        <Box id={section ? settingsSectionElementId(section) : undefined} sx={{ display: "flex", justifyContent: "space-between", scrollMarginTop: `${headerHeight + 16}px` }}>
            <Typography variant="h6" gutterBottom>{title}</Typography>
            <Box>
                <Button
                    color="info"
                    variant="outlined"
                    onClick={onToggle}
                >
                    {showSettings ? 'Hide' : 'Show'}
                </Button>
            </Box>
        </Box>
    );
}

export default SettingsToggleButton;
