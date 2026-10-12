// @vitest-environment-options {"url": "https://tharsis.example.com/"}
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import Markdown from './Markdown';

const external = 'https://evil.example/x?d=secret';
const sameOrigin = `${window.location.origin}/groups/a/-/runs`;

afterEach(() => {
    vi.restoreAllMocks();
});

describe('Markdown', () => {
    describe('by default (READMEs, announcements)', () => {
        it('loads external images', () => {
            render(<Markdown>{`![badge](${external})`}</Markdown>);
            expect(screen.getByRole('img')).toHaveAttribute('src', external);
        });

        it('opens external links directly in a new tab', () => {
            render(<Markdown>{`[docs](${external})`}</Markdown>);
            const link = screen.getByRole('link', { name: 'docs' });
            expect(link).toHaveAttribute('href', external);
            expect(link).toHaveAttribute('target', '_blank');
        });
    });

    describe('with restrictExternalContent (agent chat)', () => {
        it('does not load an external image, showing it as a link instead', () => {
            render(<Markdown restrictExternalContent>{`![leak](${external})`}</Markdown>);
            expect(screen.queryByRole('img')).not.toBeInTheDocument();
            expect(screen.getByRole('link', { name: 'Image: leak' })).toHaveAttribute('href', external);
        });

        it('still loads a same-origin image', () => {
            render(<Markdown restrictExternalContent>{`![diagram](${sameOrigin}/diagram.png)`}</Markdown>);
            expect(screen.getByRole('img')).toHaveAttribute('src', `${sameOrigin}/diagram.png`);
        });

        it('opens a same-origin link directly', () => {
            render(<Markdown restrictExternalContent>{`[run](${sameOrigin})`}</Markdown>);
            expect(screen.getByRole('link', { name: 'run' })).toHaveAttribute('target', '_blank');
        });

        it('asks before opening an external link, and opens it only once confirmed', async () => {
            render(<Markdown restrictExternalContent>{`[full error log](${external})`}</Markdown>);

            await userEvent.click(screen.getByRole('link', { name: 'full error log' }));
            expect(screen.getByText(external)).toBeInTheDocument();

            // The confirm action is a native link, so the browser opens it without popup blocking.
            const confirm = screen.getByRole('link', { name: 'Open link' });
            expect(confirm).toHaveAttribute('href', external);
            expect(confirm).toHaveAttribute('target', '_blank');
            expect(confirm).toHaveAttribute('rel', 'noopener noreferrer');

            await userEvent.click(confirm);
            await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
        });

        it('opens a link to a safe external host directly', () => {
            const docs = 'https://tharsis.martian-cloud.io/docs/guides/overviews/managed_identities';
            render(<Markdown restrictExternalContent>{`[managed identities](${docs})`}</Markdown>);
            const link = screen.getByRole('link', { name: 'managed identities' });
            expect(link).toHaveAttribute('href', docs);
            expect(link).toHaveAttribute('target', '_blank');
        });

        it.each([
            ['a subdomain of a safe host', 'https://evil.tharsis.martian-cloud.io/x?d=secret'],
            ['a host that ends with a safe host', 'https://notregistry.terraform.io/x?d=secret'],
            ['a safe host on another port', 'https://registry.terraform.io:8443/x?d=secret'],
        ])('asks before opening %s', async (_, href) => {
            render(<Markdown restrictExternalContent>{`[link](${href})`}</Markdown>);

            await userEvent.click(screen.getByRole('link', { name: 'link' }));
            expect(screen.getByRole('dialog')).toBeInTheDocument();
            expect(screen.getByRole('link', { name: 'Open link' })).toBeInTheDocument();
        });

        it('still does not load an image from a safe host', () => {
            // The list covers links only: an image is fetched without a click.
            render(<Markdown restrictExternalContent>{`![logo](https://registry.terraform.io/logo.png)`}</Markdown>);
            expect(screen.queryByRole('img')).not.toBeInTheDocument();
        });

        it('closes without offering the link when the confirmation is cancelled', async () => {
            render(<Markdown restrictExternalContent>{`[full error log](${external})`}</Markdown>);

            await userEvent.click(screen.getByRole('link', { name: 'full error log' }));
            await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
            expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
            expect(screen.queryByRole('link', { name: 'Open link' })).not.toBeInTheDocument();
        });
    });
});
