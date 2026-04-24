import CloseIcon from '@mui/icons-material/Close';
import FilterListIcon from '@mui/icons-material/FilterList';
import { Typography } from '@mui/material';
import Box from '@mui/material/Box';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import IconButton from '@mui/material/IconButton';
import Popover from '@mui/material/Popover';
import Toolbar from '@mui/material/Toolbar';
import { useContext, useState } from 'react';
import { ErrorBoundary } from 'react-error-boundary';
import { UserContext } from '../UserContext';
import { useAgentCopilot } from './AgentCopilotProvider';
import AgentSessionChat from './AgentSessionChat';
import { AgentSessionRuntimeProvider } from './AgentSessionRuntimeProvider';
import { AgentSessionTraceViewerButton } from './AgentSessionTraceViewerDialog';
import MartianAgentIcon from './MartianAgentIcon';

function SidebarToolbar({ onClose, showToolCalls, onToggleToolCalls }: { onClose: () => void; showToolCalls: boolean; onToggleToolCalls: () => void }) {
    const { agentSessionId } = useAgentCopilot();
    const user = useContext(UserContext);
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);

    return (
        <Toolbar variant="dense" sx={{
            justifyContent: 'flex-end',
            alignItems: 'center',
            minHeight: 32,
            background: theme => {
                const main = '#121212';
                const light = theme.lighten(main, 0.1);
                const dark = theme.darken(main, 0.3);
                return `linear-gradient(135deg, ${main}, ${dark}, ${light}, ${dark})`;
            },
            backgroundSize: '200% 200%',
            animation: 'gradientShift 3s ease forwards',
            '@keyframes gradientShift': {
                '0%': { backgroundPosition: '0% 50%' },
                '100%': { backgroundPosition: '100% 50%' },
            },
            color: 'common.white',
        }}>
            <MartianAgentIcon sx={{ color: 'common.white', fontSize: 20 }} />
            <Typography variant="subtitle1" sx={{ flexGrow: 1, color: 'common.white', fontWeight: 600 }} ml={1}>Copilot</Typography>
            {user.admin && agentSessionId && <AgentSessionTraceViewerButton sessionId={agentSessionId} />}
            <IconButton onClick={e => setAnchorEl(e.currentTarget)} size="medium" sx={{ color: 'common.white' }}>
                <FilterListIcon />
            </IconButton>
            <Popover
                open={Boolean(anchorEl)}
                anchorEl={anchorEl}
                onClose={() => setAnchorEl(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
                <Box sx={{ p: 1.5 }}>
                    <FormControlLabel
                        control={<Checkbox size="small" checked={showToolCalls} onChange={onToggleToolCalls} />}
                        label="Display tools"
                        slotProps={{ typography: { variant: 'body2' } }}
                    />
                </Box>
            </Popover>
            <IconButton onClick={onClose} size="medium" sx={{ color: 'common.white' }}>
                <CloseIcon />
            </IconButton>
        </Toolbar>
    );
}

function AgentSessionChatSidebar() {
    const { sidebarWidth, expanded, togglePanel } = useAgentCopilot();
    const [showToolCalls, setShowToolCalls] = useState(true);

    if (!expanded) return null;

    return (
        <Box
            sx={{
                position: 'fixed',
                right: 0,
                top: 0,
                width: sidebarWidth,
                height: '100vh',
                display: 'flex',
                flexDirection: 'column',
                gap: 1,
                bgcolor: 'background.paper',
                borderLeft: 1,
                borderColor: 'divider',
                transition: 'width 0.2s ease',
            }}
        >

            <SidebarToolbar onClose={togglePanel} showToolCalls={showToolCalls} onToggleToolCalls={() => setShowToolCalls(v => !v)} />
            <ErrorBoundary fallbackRender={({ error }) => (
                <Box sx={{ p: 2 }}>
                    <Typography color="error" variant="body2">{error?.message ?? 'Something went wrong'}</Typography>
                </Box>
            )}>
                <AgentSessionRuntimeProvider>
                    <AgentSessionChat showToolCalls={showToolCalls} />
                </AgentSessionRuntimeProvider>
            </ErrorBoundary>
        </Box>

    );
}

export default AgentSessionChatSidebar;
