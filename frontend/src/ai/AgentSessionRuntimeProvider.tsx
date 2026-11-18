import type { AppendMessage, ThreadMessageLike } from '@assistant-ui/react';
import {
    AssistantRuntimeProvider,
    AuiConfig,
    MessageNotSentError,
    Suggestions,
    useExternalStoreRuntime,
} from '@assistant-ui/react';
import { withKey } from '@assistant-ui/tap';
import { useSnackbar } from 'notistack';
import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react';
import { useRelayEnvironment } from 'react-relay/hooks';
import { cancelAgentRun, createAgentRun, createAgentSession } from './agentSession';
import { useAgentCopilot } from './AgentCopilotProvider';
import { useAgentSessionSubscription } from './useAgentSessionSubscription';

const convertMessage = (m: ThreadMessageLike) => m;

export function AgentSessionRuntimeProvider({ children }: Readonly<{ children: ReactNode }>) {
    const environment = useRelayEnvironment();
    const { enqueueSnackbar } = useSnackbar();
    const { agentSessionId, setAgentSessionId, state: copilotState } = useAgentCopilot();
    const [messages, setMessages] = useState<ThreadMessageLike[]>([]);
    const runIdRef = useRef<string | undefined>(undefined);

    const { isRunning, awaitCancel } = useAgentSessionSubscription(agentSessionId, setMessages);

    const startRun = useCallback(async (userMessage: string) => {
        const copilotContext = copilotState?.contextMessage ? [copilotState.contextMessage] : [];
        const previousRunId = runIdRef.current;
        let sessionId = agentSessionId;
        if (!sessionId) {
            sessionId = await createAgentSession(environment);
            setAgentSessionId(sessionId);
        }
        const id = await createAgentRun(environment, sessionId, userMessage, copilotContext, previousRunId);
        runIdRef.current = id;
    }, [environment, copilotState?.contextMessage, agentSessionId, setAgentSessionId]);

    const onNew = useCallback(async (message: AppendMessage) => {
        const text = message.content[0]?.type === 'text' ? message.content[0].text : '';
        if (!text) return;
        try {
            await startRun(text);
        } catch (error) {
            const reason = error instanceof Error ? error.message : String(error);
            enqueueSnackbar(`Failed to send message: ${reason}`, { variant: 'error' });
            // The run was never created, so hand the text back to the composer instead of losing it.
            throw new MessageNotSentError(reason);
        }
    }, [startRun, enqueueSnackbar]);

    const onCancel = useCallback(async () => {
        const runId = runIdRef.current;
        if (!runId) return;

        const cancelledPromise = awaitCancel();

        try {
            await cancelAgentRun(environment, runId);
        } catch {
            // Run may have already finished
            return;
        }

        await cancelledPromise;
    }, [environment, awaitCancel]);

    const store = useMemo(() => ({
        messages,
        isRunning,
        onNew,
        onCancel,
        convertMessage,
    }), [messages, isRunning, onNew, onCancel]);

    const runtime = useExternalStoreRuntime(store);

    const defaultSuggestions = useMemo(() => [
        { title: "How to get started with Tharsis", prompt: "How do I get started with Tharsis?" },
    ], []);

    const allSuggestions = useMemo(() => [...(copilotState?.suggestions ?? []), ...defaultSuggestions].map(s => ({ ...s, label: '' })), [copilotState?.suggestions, defaultSuggestions]);
    const suggestionsKey = useMemo(() => allSuggestions.map(s => s.prompt).join('|'), [allSuggestions]);

    const config = AuiConfig({
        suggestions: withKey(suggestionsKey, Suggestions(allSuggestions)),
    });

    return (
        <AssistantRuntimeProvider runtime={runtime} config={config}>
            {children}
        </AssistantRuntimeProvider>
    );
}
