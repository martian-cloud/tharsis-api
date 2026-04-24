import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react';

type CopilotSuggestion = { title: string; prompt: string };

type CopilotState = {
    suggestions: CopilotSuggestion[];
    contextMessage: string | undefined;
 };

const COLLAPSED_WIDTH = 0;
const EXPANDED_WIDTH = 450;

type AgentCopilotContextType = {
    agentSessionId: string | undefined;
    state: CopilotState | undefined;
    setAgentSessionId: (id: string | undefined) => void;
    setState: (state: CopilotState | undefined) => void;
    sidebarWidth: number;
    expanded: boolean;
    togglePanel: () => void;
};

const AgentCopilotCtx = createContext<AgentCopilotContextType>({
    agentSessionId: undefined,
    state: undefined,
    setAgentSessionId: () => { },
    setState: () => { },
    sidebarWidth: COLLAPSED_WIDTH,
    expanded: false,
    togglePanel: () => { }
});

export const useAgentCopilot = () => useContext(AgentCopilotCtx);

export function AgentCopilotProvider({ children }: Readonly<{ children: ReactNode }>) {
    const [agentSessionId, setAgentSessionId] = useState<string | undefined>(undefined);
    const [state, setState] = useState<CopilotState | undefined>(undefined);
    const [expanded, setExpanded] = useState(false);
    const theme = useTheme();
    const mobile = useMediaQuery(theme.breakpoints.down('md'));

    const togglePanel = useCallback(() => {
        setExpanded(prev => {
            if (prev) setAgentSessionId(undefined);
            return !prev;
        });
    }, []);

    const sidebarWidth = mobile ? 0 : expanded ? EXPANDED_WIDTH : COLLAPSED_WIDTH;

    const value = useMemo(() => ({
        agentSessionId,
        setAgentSessionId,
        state,
        setState,
        sidebarWidth,
        expanded,
        togglePanel,
    }), [agentSessionId, state, sidebarWidth, expanded, togglePanel]);

    return (
        <AgentCopilotCtx.Provider value={value}>
            {children}
        </AgentCopilotCtx.Provider>
    );
}
