import type { ThreadMessageLike } from '@assistant-ui/react';
import { describe, expect, it } from 'vitest';
import type { AgentSessionEvent } from './agentSession';
import { AgentSessionEventHandler } from './AgentSessionEventHandler';

const event = (fields: Record<string, unknown>) => fields as unknown as AgentSessionEvent;

// handle feeds events through a handler and returns the message list it builds.
const handle = (events: AgentSessionEvent[]) => {
    let messages: ThreadMessageLike[] = [];
    const handler = new AgentSessionEventHandler(update => { messages = update(messages); }, () => { });
    events.forEach(e => handler.handleEvent(e));
    return messages;
};

const toolCallPart = (messages: ThreadMessageLike[]) => {
    const content = messages.find(m => m.role === 'assistant')?.content;
    return Array.isArray(content) ? content.find(p => p.type === 'tool-call') : undefined;
};

const runStarted = event({ __typename: 'AgentSessionRunStartedEvent', type: 'RUN_STARTED', threadId: 's1', runId: 'r1' });
const runFinished = event({ __typename: 'AgentSessionRunFinishedEvent', type: 'RUN_FINISHED', threadId: 's1', runId: 'r1' });
const toolCallStart = (timestamp: number | null) => event({
    __typename: 'AgentSessionToolCallStartEvent', type: 'TOOL_CALL_START', toolCallId: 't1', toolCallName: 'get_run', parentMessageId: 'm1', timestamp,
});
const toolCallArgs = event({ __typename: 'AgentSessionToolCallArgsEvent', type: 'TOOL_CALL_ARGS', toolCallId: 't1', delta: '{"id":"run-1"}' });
const toolCallEnd = event({ __typename: 'AgentSessionToolCallEndEvent', type: 'TOOL_CALL_END', toolCallId: 't1' });
const toolCallResult = (timestamp: number | null) => event({
    __typename: 'AgentSessionToolCallResultEvent', type: 'TOOL_CALL_RESULT', messageId: 'm2', toolCallId: 't1', content: '{"status":"applied"}', role: 'tool', timestamp,
});

describe('AgentSessionEventHandler tool call timing', () => {
    it('sets the start and end from the server timestamps of a finished tool call', () => {
        const messages = handle([
            runStarted, toolCallStart(1_763_500_000_000), toolCallArgs, toolCallEnd, toolCallResult(1_763_500_002_500), runFinished,
        ]);

        expect(toolCallPart(messages)).toMatchObject({
            toolCallId: 't1',
            result: { status: 'applied' },
            timing: { startedAt: 1_763_500_000_000, completedAt: 1_763_500_002_500 },
        });
    });

    it('sets only the start while the tool call has no result', () => {
        const part = toolCallPart(handle([runStarted, toolCallStart(1_763_500_000_000), toolCallArgs, toolCallEnd]));

        expect(part).toMatchObject({ timing: { startedAt: 1_763_500_000_000 } });
        expect(part).not.toHaveProperty('timing.completedAt');
    });

    it('leaves timing off when the events have no timestamps', () => {
        const part = toolCallPart(handle([runStarted, toolCallStart(null), toolCallArgs, toolCallEnd, toolCallResult(null), runFinished]));

        expect(part).toMatchObject({ toolCallId: 't1', result: { status: 'applied' } });
        expect(part).not.toHaveProperty('timing');
    });
});
