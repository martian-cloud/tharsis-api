import graphql from 'babel-plugin-relay/macro';
import type { IEnvironment } from 'relay-runtime';
import { commitMutation, requestSubscription } from 'relay-runtime';
import type { agentSessionEventsSubscription$data } from './__generated__/agentSessionEventsSubscription.graphql';

export type AgentSessionEvent = agentSessionEventsSubscription$data['agentSessionEvents'];
export type StreamSubscription = { unsubscribe(): void };

const CreateAgentSessionMutation = graphql`
    mutation agentSessionCreateSessionMutation($input: CreateAgentSessionInput!) {
        createAgentSession(input: $input) {
            session {
                id
            }
            problems {
                message
                type
            }
        }
    }
`;

const CreateAgentRunMutation = graphql`
    mutation agentSessionCreateRunMutation($input: CreateAgentRunInput!) {
        createAgentRun(input: $input) {
            run {
                id
                status
            }
            problems {
                message
                type
            }
        }
    }
`;

const CancelAgentRunMutation = graphql`
    mutation agentSessionCancelRunMutation($input: CancelAgentRunInput!) {
        cancelAgentRun(input: $input) {
            run {
                id
                status
            }
            problems {
                message
                type
            }
        }
    }
`;

const AgentSessionEventsSubscription = graphql`
    subscription agentSessionEventsSubscription($input: AgentSessionEventSubscriptionInput!) {
        agentSessionEvents(input: $input) {
            __typename
            ... on AgentSessionRunStartedEvent { type threadId runId }
            ... on AgentSessionRunFinishedEvent { type threadId runId }
            ... on AgentSessionRunErrorEvent { type threadId runId message }
            ... on AgentSessionCustomEvent { type name value }
            ... on AgentSessionTextMessageStartEvent { type messageId role }
            ... on AgentSessionTextMessageContentEvent { type messageId delta }
            ... on AgentSessionTextMessageEndEvent { type messageId }
            ... on AgentSessionToolCallStartEvent { type toolCallId toolCallName parentMessageId }
            ... on AgentSessionToolCallArgsEvent { type toolCallId delta }
            ... on AgentSessionToolCallEndEvent { type toolCallId }
            ... on AgentSessionToolCallResultEvent { type messageId toolCallId content role }
            ... on AgentSessionReasoningStartEvent { type messageId }
            ... on AgentSessionReasoningMessageStartEvent { type messageId role }
            ... on AgentSessionReasoningMessageContentEvent { type messageId delta }
            ... on AgentSessionReasoningMessageEndEvent { type messageId }
            ... on AgentSessionReasoningEndEvent { type messageId }
            ... on AgentSessionStepStartedEvent { type stepName }
            ... on AgentSessionStepFinishedEvent { type stepName }
        }
    }
`;

/** Create a new agent session via GraphQL mutation. Returns the session ID. */
export function createAgentSession(environment: IEnvironment): Promise<string> {
    return new Promise((resolve, reject) => {
        commitMutation(environment, {
            mutation: CreateAgentSessionMutation,
            variables: { input: {} },
            onCompleted: (response: any, errors) => {
                if (errors?.length) {
                    reject(new Error(errors[0].message));
                    return;
                }
                const payload = response.createAgentSession;
                if (payload.problems?.length) {
                    reject(new Error(payload.problems[0].message));
                    return;
                }
                resolve(payload.session.id);
            },
            onError: reject,
        });
    });
}

/** Create a new agent run via GraphQL mutation. Returns the run ID. */
export function createAgentRun(
    environment: IEnvironment,
    sessionId: string,
    message: string,
    context?: string[],
    previousRunId?: string,
): Promise<string> {
    return new Promise((resolve, reject) => {
        commitMutation(environment, {
            mutation: CreateAgentRunMutation,
            variables: { input: { sessionId, message, context, previousRunId } },
            onCompleted: (response: any, errors) => {
                if (errors?.length) {
                    reject(new Error(errors[0].message));
                    return;
                }
                const payload = response.createAgentRun;
                if (payload.problems?.length) {
                    reject(new Error(payload.problems[0].message));
                    return;
                }
                resolve(payload.run.id);
            },
            onError: reject,
        });
    });
}

/** Cancel an in-progress agent run via GraphQL mutation. */
export function cancelAgentRun(
    environment: IEnvironment,
    runId: string,
): Promise<void> {
    return new Promise((resolve, reject) => {
        commitMutation(environment, {
            mutation: CancelAgentRunMutation,
            variables: { input: { runId } },
            onCompleted: (response: any, errors) => {
                if (errors?.length) {
                    reject(new Error(errors[0].message));
                    return;
                }
                const payload = response.cancelAgentRun;
                if (payload.problems?.length) {
                    reject(new Error(payload.problems[0].message));
                    return;
                }
                resolve();
            },
            onError: reject,
        });
    });
}

/** Subscribe to agent session events via GraphQL subscription. */
export function subscribeToAgentSession(
    environment: IEnvironment,
    sessionId: string,
    onEvent: (event: AgentSessionEvent) => void,
    onError?: (err: Error) => void,
): StreamSubscription {
    const { dispose } = requestSubscription(environment, {
        subscription: AgentSessionEventsSubscription,
        variables: { input: { sessionId } },
        onNext: (response) => {
            const data = response as agentSessionEventsSubscription$data | null | undefined;
            if (data?.agentSessionEvents) {
                onEvent(data.agentSessionEvents);
            }
        },
        onError: (err: Error) => {
            console.error('Agent session subscription error:', err);
            onError?.(err);
        }
    });

    return { unsubscribe: dispose };
}
