import { ThemeProvider } from '@mui/material/styles';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { SnackbarProvider } from 'notistack';
import { RelayEnvironmentProvider } from 'react-relay/hooks';
import { Environment, Network, RecordSource, Store } from 'relay-runtime';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import theme from '../theme';
import AgentSessionChat from './AgentSessionChat';
import { AgentSessionRuntimeProvider } from './AgentSessionRuntimeProvider';
import { createAgentSession } from './agentSession';

vi.mock('./agentSession', () => ({
    createAgentSession: vi.fn(),
    createAgentRun: vi.fn(),
    cancelAgentRun: vi.fn(),
    subscribeToAgentSession: vi.fn(() => ({ unsubscribe: () => { } })),
}));

// The mutations are mocked, so nothing is ever fetched from this environment.
const environment = new Environment({
    network: Network.create(() => Promise.reject(new Error('no network in tests'))),
    store: new Store(new RecordSource()),
});

describe('AgentSessionRuntimeProvider', () => {
    // jsdom has no ResizeObserver, which the thread viewport uses to follow its content.
    beforeAll(() => {
        vi.stubGlobal('ResizeObserver', class {
            observe() { }
            unobserve() { }
            disconnect() { }
        });
    });

    it('puts the message back in the composer and shows the error when the send fails', async () => {
        vi.mocked(createAgentSession).mockRejectedValue(new Error('AI features are disabled'));

        render(
            <RelayEnvironmentProvider environment={environment}>
                <ThemeProvider theme={theme}>
                    <SnackbarProvider>
                        <AgentSessionRuntimeProvider>
                            <AgentSessionChat />
                        </AgentSessionRuntimeProvider>
                    </SnackbarProvider>
                </ThemeProvider>
            </RelayEnvironmentProvider>
        );

        const input = screen.getByPlaceholderText('Ask something…');
        fireEvent.change(input, { target: { value: 'Why did my run fail?' } });
        fireEvent.keyDown(input, { key: 'Enter' });

        expect(await screen.findByText('Failed to send message: AI features are disabled')).toBeInTheDocument();
        await waitFor(() => expect(input).toHaveValue('Why did my run fail?'));
        expect(createAgentSession).toHaveBeenCalledTimes(1);
    });
});
