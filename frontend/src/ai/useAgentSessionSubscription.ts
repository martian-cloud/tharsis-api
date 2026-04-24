import { useCallback, useEffect, useRef, useState } from 'react';
import { useRelayEnvironment } from 'react-relay/hooks';
import { subscribeToAgentSession, type StreamSubscription } from './agentSession';
import { AgentSessionEventHandler, type SetMessages } from './AgentSessionEventHandler';

/**
 * Manages the Relay subscription for an agent session.
 *
 * - Creates the subscription reactively when `agentSessionId` is set.
 * - Disposes the subscription when the session changes or on unmount.
 */
export function useAgentSessionSubscription(
    agentSessionId: string | undefined,
    setMessages: SetMessages,
): { isRunning: boolean; awaitCancel: () => Promise<void> } {
    const environment = useRelayEnvironment();
    const subscriptionRef = useRef<StreamSubscription | null>(null);
    const runSessionRef = useRef<AgentSessionEventHandler | null>(null);
    const [isRunning, setIsRunning] = useState(false);

    useEffect(() => {
        if (!agentSessionId) return;

        const session = new AgentSessionEventHandler(setMessages, setIsRunning);
        runSessionRef.current = session;

        subscriptionRef.current = subscribeToAgentSession(
            environment,
            agentSessionId!,
            (event) => session.handleEvent(event),
            (err: Error) => {
                console.error('Agent session subscription error:', err);
            },
        );

        return () => {
            subscriptionRef.current?.unsubscribe();
            subscriptionRef.current = null;
            runSessionRef.current = null;
        };
    }, [agentSessionId, environment, setMessages]);

    const awaitCancel = useCallback(() => {
        const session = runSessionRef.current;
        if (!session) return Promise.resolve();
        return new Promise<void>(resolve => {
            session.setCancelResolve(resolve);
        });
    }, []);

    return { isRunning, awaitCancel };
}
