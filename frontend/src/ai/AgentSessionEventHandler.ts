import type { ThreadMessageLike } from '@assistant-ui/react';
import type { AgentSessionEvent } from './agentSession';

export type SetMessages = (updater: (prev: ThreadMessageLike[]) => ThreadMessageLike[]) => void;
export type SetIsRunning = (running: boolean) => void;

type RunStartedEvent = AgentSessionEvent & { __typename: 'AgentSessionRunStartedEvent' };
type RunErrorEvent = AgentSessionEvent & { __typename: 'AgentSessionRunErrorEvent' };
type TextMessageStartEvent = AgentSessionEvent & { __typename: 'AgentSessionTextMessageStartEvent' };
type TextMessageContentEvent = AgentSessionEvent & { __typename: 'AgentSessionTextMessageContentEvent' };
type TextMessageEndEvent = AgentSessionEvent & { __typename: 'AgentSessionTextMessageEndEvent' };
type ReasoningMessageContentEvent = AgentSessionEvent & { __typename: 'AgentSessionReasoningMessageContentEvent' };
type ToolCallStartEvent = AgentSessionEvent & { __typename: 'AgentSessionToolCallStartEvent' };
type ToolCallArgsEvent = AgentSessionEvent & { __typename: 'AgentSessionToolCallArgsEvent' };
type ToolCallResultEvent = AgentSessionEvent & { __typename: 'AgentSessionToolCallResultEvent' };

type ToolCallState = {
    toolCallId: string;
    toolCallName: string;
    argsText: string;
    parsedArgs: Record<string, any> | undefined;
    result: unknown;
    isError: boolean | undefined;
};

type PartOrder =
    | { kind: 'text'; key: string }
    | { kind: 'reasoning' }
    | { kind: 'tool-call'; toolCallId: string };

type AssistantStatus =
    | { type: 'running' }
    | { type: 'complete'; reason: string }
    | { type: 'requires-action'; reason: 'tool-calls' }
    | { type: 'incomplete'; reason: 'error' | 'cancelled' };

/**
 * AgentSessionEventHandler processes AG-UI events from a session subscription and maintains
 * the ThreadMessageLike[] list for the external store runtime.
 */
export class AgentSessionEventHandler {
    // Current assistant message state (reset per run)
    private assistantId: string | undefined;
    private assistantAdded = false;
    private status: AssistantStatus | undefined;
    private textParts = new Map<string, { buffer: string; touched: boolean }>();
    private activeTextMessageId: string | undefined;
    private reasoningBuffer = '';
    private reasoningActive = false;
    private hasReasoningPart = false;
    private toolCalls = new Map<string, ToolCallState>();
    private partOrder: PartOrder[] = [];
    private textPartCounter = 0;

    // Active run ID for tracking completions
    private currentRunId: string | undefined;

    // Active user message being streamed
    private activeUserMsgId: string | undefined;

    // Track completed IDs to discard duplicates on subscription reconnect
    private completedRunIds = new Set<string>();
    private completedUserMessageIds = new Set<string>();
    private completedToolCallIds = new Set<string>();
    private skippingRun = false;
    private skippingUserMsg = false;
    private skippingToolCall = false;

    // Resolve function for cancel promise — called when RUN_CANCELLED is processed
    private cancelResolve: (() => void) | undefined;

    constructor(
        private setMessages: SetMessages,
        private setIsRunning: SetIsRunning,
    ) { }

    /** Set a resolve callback that will be called when a RUN_CANCELLED event is processed. */
    setCancelResolve(resolve: (() => void) | undefined): void {
        this.cancelResolve = resolve;
    }

    handleEvent(event: AgentSessionEvent): void {
        if (!('type' in event)) return;
        switch (event.type) {
            case 'RUN_STARTED':
                return this.onRunStarted(event as AgentSessionEvent & { __typename: 'AgentSessionRunStartedEvent' });
            case 'RUN_FINISHED':
                if (this.skippingRun) { this.skippingRun = false; return; }
                return this.onRunFinished();
            case 'RUN_ERROR':
                if (this.skippingRun) { this.skippingRun = false; return; }
                return this.onRunError(event as AgentSessionEvent & { __typename: 'AgentSessionRunErrorEvent' });
            case 'CUSTOM':
                if ('name' in event && event.name === 'RUN_CANCELLED') {
                    if (this.skippingRun) { this.skippingRun = false; return; }
                    return this.onRunCancelled();
                }
                return;
        }

        if (this.skippingRun) return;

        switch (event.type) {
            case 'TEXT_MESSAGE_START':
                return this.onTextMessageStart(event as AgentSessionEvent & { __typename: 'AgentSessionTextMessageStartEvent' });
            case 'TEXT_MESSAGE_CONTENT':
                if (this.skippingUserMsg) return;
                return this.onTextMessageContent(event as AgentSessionEvent & { __typename: 'AgentSessionTextMessageContentEvent' });
            case 'TEXT_MESSAGE_END':
                if (this.skippingUserMsg) { this.skippingUserMsg = false; return; }
                return this.onTextMessageEnd(event as AgentSessionEvent & { __typename: 'AgentSessionTextMessageEndEvent' });
            case 'REASONING_START':
            case 'REASONING_MESSAGE_START':
                return this.onReasoningStart();
            case 'REASONING_MESSAGE_CONTENT':
                return this.onReasoningContent(event as AgentSessionEvent & { __typename: 'AgentSessionReasoningMessageContentEvent' });
            case 'REASONING_MESSAGE_END':
            case 'REASONING_END':
                return this.onReasoningEnd();
            case 'TOOL_CALL_START':
                return this.onToolCallStart(event as AgentSessionEvent & { __typename: 'AgentSessionToolCallStartEvent' });
            case 'TOOL_CALL_ARGS':
                if (this.skippingToolCall) return;
                return this.onToolCallArgs(event as AgentSessionEvent & { __typename: 'AgentSessionToolCallArgsEvent' });
            case 'TOOL_CALL_END':
                if (this.skippingToolCall) return;
                return this.onToolCallEnd();
            case 'TOOL_CALL_RESULT':
                if (this.skippingToolCall) { this.skippingToolCall = false; return; }
                return this.onToolCallResult(event as AgentSessionEvent & { __typename: 'AgentSessionToolCallResultEvent' });
        }
    }

    // --- Lifecycle ---

    private onRunStarted(event: RunStartedEvent): void {
        const runId = event.runId as string | undefined;
        if (runId && this.completedRunIds.has(runId)) {
            this.skippingRun = true;
            return;
        }
        this.skippingRun = false;

        // Reset aggregator state for new run
        this.textParts.clear();
        this.reasoningBuffer = '';
        this.reasoningActive = false;
        this.hasReasoningPart = false;
        this.toolCalls.clear();
        this.partOrder = [];
        this.textPartCounter = 0;
        this.activeTextMessageId = undefined;
        this.activeUserMsgId = undefined;

        // Prepare assistant message ID but don't add it yet — wait for first assistant content
        this.currentRunId = event.runId as string | undefined;
        this.assistantId = `assistant-${event.runId ?? Date.now()}`;
        this.assistantAdded = false;
        this.status = { type: 'running' };

        this.setIsRunning(true);
    }

    private completeCurrentRun(): void {
        if (this.currentRunId) this.completedRunIds.add(this.currentRunId);
    }

    private onRunFinished(): void {
        this.completeCurrentRun();
        const hasUnresolved = Array.from(this.toolCalls.values()).some(
            tc => tc.result === undefined,
        );
        this.status = hasUnresolved
            ? { type: 'requires-action', reason: 'tool-calls' }
            : { type: 'complete', reason: 'unknown' };
        this.syncAssistant();
        this.setIsRunning(false);
    }

    private onRunError(event: RunErrorEvent): void {
        this.completeCurrentRun();
        if (!this.assistantId) {
            this.assistantId = `assistant-error-${Date.now()}`;
        }

        if (event.message) {
            const id = this.resolveTextMessageId(undefined);
            this.appendText(id, event.message);
        }

        this.status = { type: 'incomplete', reason: 'error' };
        this.syncAssistant();
        this.setIsRunning(false);
    }

    private onRunCancelled(): void {
        this.completeCurrentRun();
        this.status = { type: 'incomplete', reason: 'cancelled' };
        this.syncAssistant();
        this.setIsRunning(false);

        if (this.cancelResolve) {
            this.cancelResolve();
            this.cancelResolve = undefined;
        }
    }

    // --- Text messages ---

    private onTextMessageStart(event: TextMessageStartEvent): void {
        if (event.role === 'user') {
            if (event.messageId && this.completedUserMessageIds.has(event.messageId)) {
                this.skippingUserMsg = true;
                return;
            }
            this.skippingUserMsg = false;

            this.activeUserMsgId = event.messageId;
            // Replace optimistic user message if it exists, otherwise add new
            this.setMessages(prev => {
                return [...prev, {
                    role: 'user' as const,
                    id: event.messageId,
                    content: [{ type: 'text' as const, text: '' }],
                }];
            });
            return;
        }

        // Assistant text
        this.activeUserMsgId = undefined;
        const id = this.startTextMessage(event.messageId);
        this.markTextPartTouched(id);
        this.syncAssistant();
    }

    private onTextMessageContent(event: TextMessageContentEvent): void {
        const delta = event.delta ?? '';
        if (!delta) return;

        // If we're streaming a user message
        if (this.activeUserMsgId) {
            const uid = this.activeUserMsgId;
            this.setMessages(prev => prev.map(m => {
                if (m.id !== uid) return m;
                const existing = Array.isArray(m.content) && m.content[0]?.type === 'text'
                    ? m.content[0].text : '';
                return { ...m, content: [{ type: 'text' as const, text: existing + delta }] };
            }));
            return;
        }

        // Assistant text
        const id = this.resolveTextMessageId(event.messageId);
        this.appendText(id, delta);
        this.syncAssistant();
    }

    private onTextMessageEnd(event: TextMessageEndEvent): void {
        if (this.activeUserMsgId) {
            if (this.activeUserMsgId) this.completedUserMessageIds.add(this.activeUserMsgId);
            this.activeUserMsgId = undefined;
            return;
        }
        if (event.messageId && this.activeTextMessageId === event.messageId) {
            this.activeTextMessageId = undefined;
        }
    }

    // --- Reasoning ---

    private onReasoningStart(): void {
        this.reasoningActive = true;
        this.ensureReasoningPart();
        this.syncAssistant();
    }

    private onReasoningContent(event: ReasoningMessageContentEvent): void {
        const delta = event.delta ?? '';
        if (!delta) return;
        this.reasoningBuffer += delta;
        this.ensureReasoningPart();
        this.syncAssistant();
    }

    private onReasoningEnd(): void {
        this.reasoningActive = false;
        this.syncAssistant();
    }

    // --- Tool calls ---

    private onToolCallStart(event: ToolCallStartEvent): void {
        this.activeTextMessageId = undefined;
        const id = event.toolCallId;
        if (!id) return;
        if (this.completedToolCallIds.has(id)) {
            this.skippingToolCall = true;
            return;
        }
        this.skippingToolCall = false;

        if (!this.partOrder.some(p => p.kind === 'tool-call' && p.toolCallId === id)) {
            this.partOrder.push({ kind: 'tool-call', toolCallId: id });
        }
        this.toolCalls.set(id, {
            toolCallId: id,
            toolCallName: event.toolCallName ?? 'tool',
            argsText: '',
            parsedArgs: undefined,
            result: undefined,
            isError: undefined,
        });
        this.syncAssistant();
    }

    private onToolCallArgs(event: ToolCallArgsEvent): void {
        const delta = event.delta ?? '';
        if (!delta || !event.toolCallId) return;

        const entry = this.toolCalls.get(event.toolCallId);
        if (!entry) return;

        entry.argsText += delta;
        try {
            const parsed = JSON.parse(entry.argsText);
            if (parsed && typeof parsed === 'object') {
                entry.parsedArgs = parsed;
            }
        } catch {
            // incomplete JSON, will parse on next append or on end
        }
        this.syncAssistant();
    }

    private onToolCallEnd(): void {
        this.syncAssistant();
    }

    private onToolCallResult(event: ToolCallResultEvent): void {
        const id = event.toolCallId;
        if (!id) return;

        let entry = this.toolCalls.get(id);
        if (!entry) {
            // Result arrived for a tool call we haven't seen start for (e.g. replay)
            entry = {
                toolCallId: id,
                toolCallName: 'tool',
                argsText: '',
                parsedArgs: undefined,
                result: undefined,
                isError: undefined,
            };
            this.toolCalls.set(id, entry);
            if (!this.partOrder.some(p => p.kind === 'tool-call' && p.toolCallId === id)) {
                this.partOrder.push({ kind: 'tool-call', toolCallId: id });
            }
        }

        entry.result = tryParseJSON(event.content ?? '');
        this.completedToolCallIds.add(id);
        this.syncAssistant();
    }

    // --- Text part management (ported from RunAggregator) ---

    private startTextMessage(messageId?: string): string {
        const id = messageId ?? this.generateTextKey();
        this.ensureTextPart(id);
        this.activeTextMessageId = id;
        return id;
    }

    private resolveTextMessageId(messageId?: string): string {
        if (messageId) {
            this.ensureTextPart(messageId);
            this.activeTextMessageId = messageId;
            return messageId;
        }
        if (this.activeTextMessageId) {
            return this.activeTextMessageId;
        }
        const generated = this.generateTextKey();
        this.ensureTextPart(generated);
        this.activeTextMessageId = generated;
        return generated;
    }

    private generateTextKey(): string {
        this.textPartCounter += 1;
        return `text-${this.textPartCounter}`;
    }

    private ensureTextPart(id: string): void {
        if (!this.textParts.has(id)) {
            this.textParts.set(id, { buffer: '', touched: false });
            if (!this.partOrder.some(p => p.kind === 'text' && p.key === id)) {
                this.partOrder.push({ kind: 'text', key: id });
            }
        }
    }

    private markTextPartTouched(id: string): void {
        const entry = this.textParts.get(id);
        if (entry) entry.touched = true;
    }

    private appendText(id: string, delta: string): void {
        this.ensureTextPart(id);
        const entry = this.textParts.get(id)!;
        entry.buffer += delta;
        entry.touched = true;
    }

    // --- Reasoning part management (ported from RunAggregator) ---

    private ensureReasoningPart(): void {
        if (this.hasReasoningPart) return;
        // Insert reasoning before the first text part
        const textIndex = this.partOrder.findIndex(p => p.kind === 'text');
        if (textIndex === -1) {
            this.partOrder.push({ kind: 'reasoning' });
        } else {
            this.partOrder.splice(textIndex, 0, { kind: 'reasoning' });
        }
        this.hasReasoningPart = true;
    }

    // --- Emit / sync ---

    private syncAssistant(): void {
        if (!this.assistantId) return;
        const msg = this.buildAssistantMessage();
        if (!this.assistantAdded) {
            this.assistantAdded = true;
            this.setMessages(prev => [...prev, msg]);
        } else {
            this.setMessages(prev => prev.map(m => m.id === this.assistantId ? msg : m));
        }
    }

    private buildAssistantMessage(): ThreadMessageLike {
        const content: NonNullable<Exclude<ThreadMessageLike['content'], string>>[number][] = [];

        for (const part of this.partOrder) {
            if (part.kind === 'reasoning') {
                if (this.reasoningActive || this.reasoningBuffer.length > 0) {
                    content.push({
                        type: 'reasoning',
                        text: this.reasoningBuffer,
                    });
                }
                continue;
            }

            if (part.kind === 'text') {
                const entry = this.textParts.get(part.key);
                if (entry?.touched) {
                    content.push({ type: 'text', text: entry.buffer });
                }
                continue;
            }

            // tool-call
            const tc = this.toolCalls.get(part.toolCallId);
            if (!tc) continue;
            content.push({
                type: 'tool-call',
                toolCallId: tc.toolCallId,
                toolName: tc.toolCallName,
                args: tc.parsedArgs ?? {},
                argsText: tc.argsText,
                ...(tc.result !== undefined ? { result: tc.result } : {}),
                ...(tc.isError !== undefined ? { isError: tc.isError } : {}),
            });
        }

        return {
            role: 'assistant',
            id: this.assistantId!,
            content,
            status: this.status ?? { type: 'running' },
        } as ThreadMessageLike;
    }
}

function tryParseJSON(value: string): unknown {
    if (!value) return value;
    try {
        return JSON.parse(value);
    } catch {
        return value;
    }
}
