import { Box, Chip, Link, Tooltip } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React, { useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { RunAnnotationsFragment_run$key } from './__generated__/RunAnnotationsFragment_run.graphql';

const monoFontFamily = 'ui-monospace,SFMono-Regular,SF Mono,Menlo,Consolas,Liberation Mono,monospace !important';

// safeHref returns the link only when it is an http(s) URL, so a malformed or javascript: link is
// never rendered as a clickable chip. Anything else falls back to a non-clickable chip.
function safeHref(link: string | null | undefined): string | undefined {
    if (!link) {
        return undefined;
    }
    try {
        const url = new URL(link);
        return url.protocol === 'http:' || url.protocol === 'https:' ? link : undefined;
    } catch {
        return undefined;
    }
}

interface Props {
    fragmentRef: RunAnnotationsFragment_run$key;
    // emptyPlaceholder, when set, is rendered (e.g. "--") instead of nothing when the run has no
    // annotations. Used in the runs table so the Annotations column is never blank.
    emptyPlaceholder?: string;
    // maxVisible, when set, caps how many annotation chips are shown initially. Excess annotations
    // are collapsed behind a "+N more" chip that expands inline on click. Unset means show all.
    maxVisible?: number;
    // stacked, when true, lays the chips out vertically (one "key: value" per line) instead of
    // wrapping them horizontally. Used by the run details sidebar; list rows leave it unset.
    stacked?: boolean;
}

// RunAnnotations renders a run's annotations as chips. A chip with a (safe http/https) link is
// clickable and opens in a new tab; one without a link is plain text. Renders nothing (or the
// emptyPlaceholder) when the run has no annotations. When maxVisible is set, shows the first N
// chips with a "+N more" toggle to expand the rest inline. Callers own any surrounding heading.
function RunAnnotations({ fragmentRef, emptyPlaceholder, maxVisible, stacked }: Props) {
    const [expanded, setExpanded] = useState(false);

    const run = useFragment<RunAnnotationsFragment_run$key>(
        graphql`
        fragment RunAnnotationsFragment_run on Run {
            annotations {
                key
                value
                link
            }
        }
        `, fragmentRef);

    if (run.annotations.length === 0) {
        // emptyPlaceholder (e.g. "--") keeps the runs-table column from being blank; elsewhere an
        // empty run renders nothing. Callers that own a heading guard it on annotations themselves.
        return emptyPlaceholder ? <React.Fragment>{emptyPlaceholder}</React.Fragment> : null;
    }

    const annotations = run.annotations;
    // canCollapse is true when there are more annotations than the cap allows — i.e. the "+N more" /
    // "show less" toggle is relevant. shouldCollapse is the collapsed (not-yet-expanded) sub-state.
    const canCollapse = maxVisible != null && annotations.length > maxVisible;
    const shouldCollapse = canCollapse && !expanded;
    const visibleAnnotations = shouldCollapse ? annotations.slice(0, maxVisible) : annotations;
    const hiddenCount = annotations.length - visibleAnnotations.length;

    // In the stacked (sidebar) view, chips are laid out vertically so each annotation reads as its
    // own "key: value" line. In the inline (list row) view, they wrap horizontally to stay compact.
    const renderChip = (annotation: { key: string; value: string; link: string | null | undefined }, index: number) => {
        const href = safeHref(annotation.link);
        return (
            <Tooltip title={annotation.value} key={`${annotation.key}-${index}`}>
                <Chip
                    component={href ? Link : 'div'}
                    href={href}
                    underline={href ? 'hover' : undefined}
                    target={href ? '_blank' : undefined}
                    rel={href ? 'noopener noreferrer' : undefined}
                    sx={{ mb: 0.5, mr: 0.5, cursor: href ? 'pointer' : 'default', fontFamily: monoFontFamily }}
                    size="xs"
                    label={<React.Fragment><strong>{annotation.key}</strong>: {annotation.value}</React.Fragment>}
                />
            </Tooltip>
        );
    };

    const chips = (
        <Box
            display="flex"
            flexDirection={stacked ? 'column' : 'row'}
            flexWrap={stacked ? 'nowrap' : 'wrap'}
            alignItems="flex-start"
        >
            {visibleAnnotations.map((annotation, index) => renderChip(annotation, index))}
            {shouldCollapse && (
                <Chip
                    size="xs"
                    label={`+${hiddenCount} more`}
                    onClick={() => setExpanded(true)}
                    sx={{ mb: 0.5, mr: 0.5, cursor: 'pointer', fontFamily: monoFontFamily }}
                />
            )}
            {expanded && canCollapse && (
                <Chip
                    size="xs"
                    label="show less"
                    onClick={() => setExpanded(false)}
                    sx={{ mb: 0.5, mr: 0.5, cursor: 'pointer', fontFamily: monoFontFamily, fontStyle: 'italic' }}
                    variant="outlined"
                />
            )}
        </Box>
    );

    return chips;
}

export default RunAnnotations;
