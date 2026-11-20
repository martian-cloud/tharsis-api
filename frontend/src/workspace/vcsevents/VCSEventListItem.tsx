import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Box, Chip, Collapse, IconButton, Link, Stack, TableCell, TableRow, Tooltip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo, useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import CopyButton from '../../common/CopyButton';
import { ResponsiveRow, useCardMode } from '../../common/ResponsiveTable';
import { StyledCode } from '../../common/StyledCode';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import VCSEventErrorMessage from './VCSEventErrorMessage';
import VCSEventStatusChip from './VCSEventStatusChip';
import { VCSEventListItemFragment_vcsEvent$key } from './__generated__/VCSEventListItemFragment_vcsEvent.graphql';
import { VCSProviderType } from './__generated__/VCSEventsFragment_vcsEvents.graphql';

// Lets the error panel row span the whole table.
const COLUMN_COUNT = 6;

const COMMIT_PATHS: Partial<Record<VCSProviderType, string>> = {
    github: 'commit',
    gitlab: '-/commit',
};

// Matches the prefixes the API strips before matching branch and tag filters.
const REF_PREFIXES = ['refs/heads/', 'refs/tags/'];

function trimRefPrefix(ref: string): string {
    const prefix = REF_PREFIXES.find(p => ref.startsWith(p));
    return prefix ? ref.slice(prefix.length) : ref;
}

// Only http(s) repository URLs are linked, since they come from webhook data.
function buildCommitUrl(repositoryUrl: string, commitId: string, providerType: VCSProviderType | null | undefined): string | null {
    const commitPath = providerType ? COMMIT_PATHS[providerType] : undefined;
    try {
        const { protocol } = new URL(repositoryUrl);
        if (!commitPath || (protocol !== 'http:' && protocol !== 'https:')) {
            return null;
        }
    } catch {
        return null;
    }
    return `${repositoryUrl.replace(/\/+$/, '')}/${commitPath}/${encodeURIComponent(commitId)}`;
}

interface Props {
    fragmentRef: VCSEventListItemFragment_vcsEvent$key
    providerType: VCSProviderType | null | undefined
}

function VCSEventListItem({ fragmentRef, providerType }: Props) {
    const cardMode = useCardMode();
    const [expanded, setExpanded] = useState(false);

    const data = useFragment<VCSEventListItemFragment_vcsEvent$key>(
        graphql`
        fragment VCSEventListItemFragment_vcsEvent on VCSEvent {
            id
            status
            type
            sourceReferenceName
            commitId
            errorMessage
            repositoryUrl
            metadata {
                createdAt
                trn
            }
        }
        `, fragmentRef);

    const commitUrl = useMemo(
        () => data.commitId ? buildCommitUrl(data.repositoryUrl, data.commitId, providerType) : null,
        [data.commitId, data.repositoryUrl, providerType]
    );

    const referenceName = data.sourceReferenceName ? trimRefPrefix(data.sourceReferenceName) : null;
    const shortCommitId = data.commitId?.substring(0, 7);

    const commitLabel = commitUrl ? (
        <Link
            href={commitUrl}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={`Commit ${shortCommitId} (opens in new tab)`}
            color="inherit"
            underline="hover"
            variant="code"
        >
            {shortCommitId}
        </Link>
    ) : <StyledCode>{shortCommitId}</StyledCode>;

    const toggle = data.errorMessage ? (
        <IconButton
            size="small"
            aria-expanded={expanded}
            aria-label={expanded ? 'Hide error message' : 'Show error message'}
            onClick={() => setExpanded(prev => !prev)}
        >
            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
        </IconButton>
    ) : !cardMode ? (
        // Holds the chevron's space so the TRN button lines up across rows.
        <IconButton size="small" disabled aria-hidden tabIndex={-1} sx={{ visibility: 'hidden' }}>
            <ExpandMoreIcon />
        </IconButton>
    ) : null;

    const errorPanel = data.errorMessage ? (
        <Collapse in={expanded} unmountOnExit>
            <VCSEventErrorMessage message={data.errorMessage} />
        </Collapse>
    ) : null;

    return (
        <>
            <ResponsiveRow cells={[
                { primary: true, content: <VCSEventStatusChip status={data.status} /> },
                { label: 'Type', content: <Chip size="small" label={data.type} /> },
                {
                    label: 'Reference',
                    content: referenceName ? (
                        <Tooltip title={referenceName !== data.sourceReferenceName ? data.sourceReferenceName : ''}>
                            <Typography variant="code" sx={{ overflowWrap: 'anywhere' }}>{referenceName}</Typography>
                        </Tooltip>
                    ) : null
                },
                {
                    label: 'Commit',
                    content: data.commitId ? (
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            {commitLabel}
                            <CopyButton data={data.commitId} toolTip={`Copy commit ID ${shortCommitId}`} />
                        </Box>
                    ) : null
                },
                { label: 'Created', content: <Timestamp variant="body2" timestamp={data.metadata.createdAt} /> },
                {
                    content: (
                        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', justifyContent: 'flex-end' }}>
                            <TRNButton trn={data.metadata.trn} size="small" />
                            {toggle}
                        </Stack>
                    )
                },
            ]} />
            {errorPanel && cardMode && errorPanel}
            {errorPanel && !cardMode && (
                <TableRow>
                    <TableCell colSpan={COLUMN_COUNT} sx={{ p: 0, border: 0 }}>{errorPanel}</TableCell>
                </TableRow>
            )}
        </>
    );
}

export default VCSEventListItem;
