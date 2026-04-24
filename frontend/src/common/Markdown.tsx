import { Box, Checkbox, Divider, TableBody, TableCell, TableHead, TableRow, useTheme } from "@mui/material";
import Link from '@mui/material/Link';
import Table from "@mui/material/Table";
import TableContainer from "@mui/material/TableContainer";
import Typography, { TypographyProps } from '@mui/material/Typography';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';
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

function isAllowedHref(href: string | undefined): boolean {
    if (!href) return false;
    try {
        const url = new URL(href, window.location.origin);
        return url.protocol === 'https:';
    } catch {
        return false;
    }
}

function MarkdownLink({ ...props }: any) {
    if (!isAllowedHref(props.href)) {
        return <>{props.children}</>;
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

export default function Markdown(props: {children: string | null | undefined;}) {
    return (
        <Box
            sx={{
                '&>:first-child': {
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
    );
}
