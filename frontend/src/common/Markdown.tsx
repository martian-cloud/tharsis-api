import { Box, Checkbox, Divider, TableBody, TableCell, TableHead, TableRow, useTheme } from "@mui/material";
import Link from '@mui/material/Link';
import Table from "@mui/material/Table";
import TableContainer from "@mui/material/TableContainer";
import Typography, { TypographyProps } from '@mui/material/Typography';
import { createContext, useContext, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';
import ConfirmationDialog from './ConfirmationDialog';
import { StyledCode } from './StyledCode';

function MarkdownParagraph({ ...props }: any) {
    return (
        <Typography
            sx={{ wordBreak: 'break-word' }}
            fontSize={'.85rem'}
            paragraph>
            {props.children}
        </Typography>
    );
}

function MarkdownHeading({ node, ...props }: any) {
    const level = node?.tagName ? parseInt(node.tagName.charAt(1)) : 1;
    let variant: TypographyProps['variant'];
    switch (level) {
        case 1:
            variant = "h4";
            break;
        case 2:
            variant = "h5";
            break;
        case 3:
            variant = "h6";
            break;
        case 4:
            variant = "subtitle1";
            break;
        case 5:
            variant = "subtitle2";
            break;
        case 6:
            variant = "subtitle2";
            break;
        default:
            variant = "h6";
            break;
    }
    return (
        <Typography
            gutterBottom
            variant={variant}
            sx={{
                mt: level <= 2 ? 3 : 2,
                mb: 1,
                fontWeight: level <= 3 ? 600 : 500,
                borderBottom: level <= 2 ? '1px solid' : 'none',
                borderColor: 'divider',
                pb: level <= 2 ? 0.5 : 0,
            }}
        >
            {props.children}
        </Typography>
    );
}

function MarkdownTable({ ...props }: any) {
    return (
        <TableContainer sx={{ margin: '16px 0', backgroundColor: prismTheme['pre[class*="language-"]'].background as string }}>
            <Table size="small">{props.children}</Table>
        </TableContainer>
    );
}

function MarkdownTableCell({ style, node, ...props }: any) {
    const isHeader = node?.tagName === 'th';
    return <TableCell sx={{
        textAlign: style?.textAlign || 'left',
        fontWeight: isHeader ? 'bold' : 'normal',
        borderBottom: '1px solid rgba(255,255,255,0.1)',
    }} {...props} />
}

function MarkdownTableRow({ ...props }: any) {
    return <TableRow sx={{ '&:last-child td': { border: 0 } }}>{props.children}</TableRow>
}

function MarkdownTableBody({ ...props }: any) {
    return <TableBody>{props.children}</TableBody>
}

function MarkdownTableHead({ ...props }: any) {
    return <TableHead sx={{ textAlign: 'left' }}>{props.children}</TableHead>
}

// When set, off-site content is treated as untrusted: external images are not loaded and external
// links open only after the user confirms the destination. Used for text written by the AI
// assistant, which can be steered by content it reads (job logs, READMEs) into writing a URL that
// carries the user's data — an image would send it the moment the message renders, a link on click.
const RestrictExternalContentContext = createContext(false);

function parseUrl(href: string | undefined): URL | null {
    if (!href) return null;
    try {
        return new URL(href, window.location.origin);
    } catch {
        return null;
    }
}

function isAllowedHref(href: string | undefined): boolean {
    return parseUrl(href)?.protocol === 'https:';
}

function isSameOrigin(href: string | undefined): boolean {
    return parseUrl(href)?.origin === window.location.origin;
}

// Hosts whose links open without the external-link confirmation. A link carries data only to whoever
// can read the requests its host receives, so a host belongs here only if its owner is trusted and it
// can't forward a request elsewhere: no open redirects, no user-hosted content on the same host.
// Matched exactly over https, without subdomains other than a leading "www.", which is stripped first.
const SAFE_EXTERNAL_LINK_HOSTS = new Set([
    'tharsis.martian-cloud.io', // Tharsis documentation (config.docsUrl)
    'developer.hashicorp.com', // Terraform documentation
    'registry.terraform.io', // Public Terraform registry
    'github.com',
    'gitlab.com',
    'youtube.com'
]);

function isSafeExternalLink(href: string | undefined): boolean {
    const url = parseUrl(href);
    return url?.protocol === 'https:' && SAFE_EXTERNAL_LINK_HOSTS.has(url.host.replace(/^www\./, ''));
}

function ExternalLinkConfirmation({ href, children }: { href: string, children: React.ReactNode }) {
    const [open, setOpen] = useState(false);

    return (
        <>
            <Link
                color="secondary"
                underline="none"
                href={href}
                onClick={(e: React.MouseEvent) => {
                    e.preventDefault();
                    setOpen(true);
                }}>
                {children}
            </Link>
            {open && <ConfirmationDialog
                title="Open external link?"
                maxWidth="sm"
                confirmLabel="Open link"
                confirmColor="primary"
                confirmInProgress={false}
                confirmHref={href}
                onConfirm={() => setOpen(false)}
                onClose={() => setOpen(false)}
            >
                <Typography variant="body2" gutterBottom>
                    This link leaves Tharsis. Only open it if you recognize and trust the destination.
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace', wordBreak: 'break-all' }}>
                    {href}
                </Typography>
            </ConfirmationDialog>}
        </>
    );
}

function MarkdownLink({ ...props }: any) {
    const restrictExternalContent = useContext(RestrictExternalContentContext);

    if (!isAllowedHref(props.href)) {
        return <>{props.children}</>;
    }
    if (restrictExternalContent && !isSameOrigin(props.href) && !isSafeExternalLink(props.href)) {
        return <ExternalLinkConfirmation href={props.href}>{props.children}</ExternalLinkConfirmation>;
    }
    return <Link
        color="secondary"
        underline="none"
        target='_blank'
        rel='noopener noreferrer'
        href={props.href}>
        {props.children}
    </Link>
}

function MarkdownCode({ className, children, ...props }: any) {
    const match = /language-(\w+)/.exec(className || '')
    const hasNewlines = typeof children === 'string' ? children.includes('\n') : Array.isArray(children) && children.some((c: any) => typeof c === 'string' && c.includes('\n'));
    const isBlock = !!match || hasNewlines;
    return isBlock && match ? (
        <SyntaxHighlighter
            children={String(children).replace(/\n$/, '')}
            style={prismTheme}
            language={match[1]}
            wrapLongLines
            lineProps={{
                style: {
                    wordBreak: 'break-word',
                    whiteSpace: 'pre-wrap',
                    fontSize: '0.875rem'
                }
            }}
            {...props}
        />
    ) : (
        <StyledCode sx={isBlock ? { whiteSpace: 'pre-wrap', display: 'block', backgroundColor: prismTheme['pre[class*="language-"]'].background as string } : undefined}>
            {children}
        </StyledCode>
    );
}

function MarkdownImage({ ...props }: any) {
    const restrictExternalContent = useContext(RestrictExternalContentContext);

    // An off-site image is fetched as soon as it renders, so when external content is restricted it
    // is shown as a link (which goes through the external-link confirmation) instead of loaded. Without
    // alt text the label stays generic rather than the URL, which screen readers spell out; the
    // confirmation shows the URL anyway.
    if (restrictExternalContent && !isSameOrigin(props.src)) {
        return <MarkdownLink href={props.src}>{props.alt ? `Image: ${props.alt}` : 'Image (external link)'}</MarkdownLink>;
    }
    return <img style={{ maxWidth: '100%' }} {...props} />;
}

function MarkdownOrderedList({ ...props }: any) {
    return <ol style={{ paddingInlineStart: '2em', margin: '0 0 16px 0' }}>{props.children}</ol>;
}

function MarkdownUnorderedList({ ...props }: any) {
    return <ul style={{ paddingInlineStart: '2em', margin: '0 0 16px 0' }}>{props.children}</ul>;
}

function MarkdownListItem({ ...props }: any) {
    return <li style={{ marginBottom: '4px' }}>{props.children}</li>;
}

function MarkdownBlockquote({ ...props }: any) {
    const theme = useTheme();
    return (
        <Box
            component="blockquote"
            sx={{
                borderLeft: '4px solid',
                borderColor: theme.palette.mode === 'dark' ? 'grey.600' : 'grey.400',
                pl: 2,
                py: 0.5,
                my: 2,
                mx: 0,
                color: 'text.secondary',
                '& > p:last-child': { mb: 0 },
            }}
        >
            {props.children}
        </Box>
    );
}

function MarkdownHr() {
    return <Divider sx={{ my: 3 }} />;
}

function MarkdownInput({ checked, disabled, type, ...props }: any) {
    if (type === 'checkbox') {
        return (
            <Checkbox
                checked={checked}
                disabled={disabled}
                size="small"
                sx={{ p: 0, mr: 0.5, verticalAlign: 'middle' }}
                {...props}
            />
        );
    }
    return <input type={type} checked={checked} disabled={disabled} {...props} />;
}

const components = {
    h1: MarkdownHeading,
    h2: MarkdownHeading,
    h3: MarkdownHeading,
    h4: MarkdownHeading,
    h5: MarkdownHeading,
    h6: MarkdownHeading,
    p: MarkdownParagraph,
    a: MarkdownLink,
    code: MarkdownCode,
    table: MarkdownTable,
    thead: MarkdownTableHead,
    tbody: MarkdownTableBody,
    tr: MarkdownTableRow,
    td: MarkdownTableCell,
    th: MarkdownTableCell,
    img: MarkdownImage,
    ol: MarkdownOrderedList,
    ul: MarkdownUnorderedList,
    li: MarkdownListItem,
    blockquote: MarkdownBlockquote,
    hr: MarkdownHr,
    input: MarkdownInput,
};

interface MarkdownProps {
    children: string | null | undefined
    // Treat off-site images and links as untrusted (see RestrictExternalContentContext).
    restrictExternalContent?: boolean
}

export default function Markdown(props: MarkdownProps) {
    return (
        <RestrictExternalContentContext.Provider value={!!props.restrictExternalContent}>
        <Box
            sx={{
                '&>:first-of-type': {
                    marginTop: 0
                },
                '&>:last-child': {
                    marginBottom: 0
                }
            }}
        >
            <ReactMarkdown remarkPlugins={[remarkGfm]} skipHtml components={components}>
                {props.children}
            </ReactMarkdown>
        </Box>
        </RestrictExternalContentContext.Provider>
    );
}
