import { ThemeProvider } from '@mui/material/styles';
import { render as rtlRender, screen } from '@testing-library/react';
import { ReactElement } from 'react';
import { RelayEnvironmentProvider } from 'react-relay/hooks';
import { MemoryRouter } from 'react-router-dom';
import { Environment, Network, RecordSource, Store } from 'relay-runtime';
import { describe, expect, it } from 'vitest';
import theme from '../theme';
import ActorDisplay, { describeActor, trnPath } from './ActorDisplay';

// useFragment needs an environment even for a null fragment ref; nothing is ever fetched from it.
const environment = new Environment({
    network: Network.create(() => Promise.reject(new Error('no network in tests'))),
    store: new Store(new RecordSource()),
});

const render = (ui: ReactElement) => rtlRender(
    <RelayEnvironmentProvider environment={environment}>
        <ThemeProvider theme={theme}><MemoryRouter>{ui}</MemoryRouter></ThemeProvider>
    </RelayEnvironmentProvider>
);

describe('describeActor', () => {
    it('shows a user from its principal email', () => {
        expect(describeActor({ type: 'USER', id: 'U_1', userEmail: 'user@example.com' }))
            .toEqual({ text: 'user@example.com', avatar: 'gravatar', email: 'user@example.com' });
    });

    it('shows a deleted user when the user is no longer returned', () => {
        expect(describeActor({ type: 'USER', id: 'U_1' })).toEqual({ text: 'Deleted user', avatar: 'deletedUser' });
    });

    it('shows a service account from its TRN with a link to it', () => {
        expect(describeActor({ type: 'SERVICE_ACCOUNT', id: 'SA_1', trn: 'trn:service_account:top/sub/deployer' }))
            .toEqual({ text: 'top/sub/deployer', avatar: 'initial', initial: 'D', link: '/groups/top/sub/-/service_accounts/SA_1' });
    });

    it('shows a VCS provider from its TRN with a link to it', () => {
        expect(describeActor({ type: 'VCS_PROVIDER', id: 'V_1', trn: 'trn:vcs_provider:top/github' }))
            .toEqual({ text: 'top/github', avatar: 'vcsProvider', link: '/groups/top/-/vcs_providers/V_1' });
    });

    it('shows a job from its workspace path with a link to it', () => {
        expect(describeActor({ type: 'JOB', id: 'J_1', trn: 'trn:job:top/ws/J_1' }))
            .toEqual({ text: 'Job in top/ws', avatar: 'job', link: '/groups/top/ws/-/jobs/J_1' });
    });

    it('shows system and SCIM actors by name', () => {
        expect(describeActor({ type: 'SYSTEM' })).toEqual({ text: 'System', avatar: 'system' });
        expect(describeActor({ type: 'SCIM' })).toEqual({ text: 'SCIM', avatar: 'scim' });
    });

    it('describes nothing without an actor', () => {
        expect(describeActor(null)).toBeNull();
    });
});

describe('trnPath', () => {
    it.each([
        ['trn:service_account:top/sub/deployer', 'top/sub/deployer'],
        ['trn:job:top/ws/J_1', 'top/ws/J_1'],
        ['top/deployer', 'top/deployer'],
    ])('returns the path of %s', (value, expected) => {
        expect(trnPath(value)).toBe(expected);
    });
});

describe('ActorDisplay', () => {
    it('renders the placeholder as text when there is no actor', () => {
        render(<ActorDisplay variant="text" fragmentRef={null} />);
        expect(screen.getByText('--')).toBeInTheDocument();
    });

    it('renders nothing as an avatar when there is no actor', () => {
        const { container } = render(<ActorDisplay fragmentRef={null} />);
        expect(container).toBeEmptyDOMElement();
    });
});
