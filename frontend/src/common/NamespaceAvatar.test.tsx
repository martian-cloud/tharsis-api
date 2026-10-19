import { ThemeProvider } from '@mui/material/styles';
import { render as rtlRender, screen } from '@testing-library/react';
import { ReactElement } from 'react';
import { describe, expect, it } from 'vitest';
import theme from '../theme';
import NamespaceAvatar, { getNamespaceInitials } from './NamespaceAvatar';

const render = (ui: ReactElement) => rtlRender(<ThemeProvider theme={theme}>{ui}</ThemeProvider>);

describe('getNamespaceInitials', () => {
    it.each([
        ['prod-network', 'PN'],
        ['prod_db', 'PD'],
        ['my-cool-app', 'MC'],
        ['-foo', 'F'],
        ['foo--bar', 'FB'],
        ['production', 'P'],
        ['abc', 'A'],
        ['', ''],
    ])('%s => %s', (name, expected) => {
        expect(getNamespaceInitials(name)).toBe(expected);
    });
});

describe('NamespaceAvatar', () => {
    it('shows two letters for a hyphenated name', () => {
        render(<NamespaceAvatar name="prod-network" type="workspace" />);
        expect(screen.getByText('PN')).toBeInTheDocument();
    });

    it('uses the same outer box for groups and workspaces', () => {
        const { container: group } = render(<NamespaceAvatar name="a" type="group" size="lg" />);
        const { container: workspace } = render(<NamespaceAvatar name="a" type="workspace" size="lg" />);
        const groupBox = window.getComputedStyle(group.firstElementChild as Element);
        const workspaceBox = window.getComputedStyle(workspace.firstElementChild as Element);
        expect(groupBox.width).toBe(workspaceBox.width);
        expect(groupBox.height).toBe(workspaceBox.height);
    });

    it('draws a group as a tile behind the avatar and a workspace as the avatar alone', () => {
        const { container: group } = render(<NamespaceAvatar name="a" type="group" />);
        const { container: workspace } = render(<NamespaceAvatar name="a" type="workspace" />);
        expect(group.firstElementChild?.children).toHaveLength(2);
        expect(workspace.firstElementChild?.children).toHaveLength(1);
    });
});
