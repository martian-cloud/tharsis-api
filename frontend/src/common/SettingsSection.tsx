import { Collapse } from '@mui/material';
import { ReactNode, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import SettingsToggleButton, { settingsSectionElementId } from './SettingsToggleButton';

// SETTINGS_SECTION_QUERY_PARAM names the settings section to expand on load, e.g. `settings?section=drift-detection`
const SETTINGS_SECTION_QUERY_PARAM = 'section';

// useSettingsSection holds a settings section's show/hide state, which is the `section` query param: the
// section is shown while the param names it. Expanding or collapsing the section updates the param, so the
// URL can be shared or bookmarked to return directly to that section, and the section is scrolled into view
// when the page loads with it requested.
function useSettingsSection(section: string): [boolean, (show: boolean) => void] {
    const [searchParams, setSearchParams] = useSearchParams();
    const requested = searchParams.get(SETTINGS_SECTION_QUERY_PARAM) === section;

    const setShowSettings = (show: boolean) => {
        setSearchParams(prev => {
            const next = new URLSearchParams(prev);
            if (show) {
                next.set(SETTINGS_SECTION_QUERY_PARAM, section);
            } else if (next.get(SETTINGS_SECTION_QUERY_PARAM) === section) {
                next.delete(SETTINGS_SECTION_QUERY_PARAM);
            }
            return next;
        }, { replace: true });
    };

    // Scroll only when the page loads with this section requested. Expanding a section later also sets
    // the query param, so whether to scroll is decided once, from the param as it was on mount.
    const [scrollOnMount] = useState<boolean>(requested);
    useEffect(() => {
        if (scrollOnMount) {
            document.getElementById(settingsSectionElementId(section))?.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
    }, [scrollOnMount, section]);

    return [requested, setShowSettings];
}

interface Props {
    title: string;
    // section is the key used to expand this section from the `section` query param
    section: string;
    children: ReactNode;
    // unmountOnExit removes the content while the section is hidden. Defaults to true.
    unmountOnExit?: boolean;
}

// SettingsSection is a settings section's header with its Show/Hide button and the collapsible content
// below it. It owns whether the section is shown, including expanding it when the page is opened with
// the section named in the `section` query param.
function SettingsSection({ title, section, children, unmountOnExit = true }: Props) {
    const [showSettings, setShowSettings] = useSettingsSection(section);

    return (
        <>
            <SettingsToggleButton
                section={section}
                title={title}
                showSettings={showSettings}
                onToggle={() => setShowSettings(!showSettings)}
            />
            <Collapse
                in={showSettings}
                timeout="auto"
                unmountOnExit={unmountOnExit}
            >
                {children}
            </Collapse>
        </>
    );
}

export default SettingsSection;
