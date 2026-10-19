import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Box, Chip, Stack, Tooltip, Typography } from '@mui/material';
import Link from '@mui/material/Link';
import ListItem from '@mui/material/ListItem';
import ListItemButton from '@mui/material/ListItemButton';
import { useTheme } from '@mui/material/styles';
import graphql from 'babel-plugin-relay/macro';
import React, { Suspense, useState } from 'react';
import { useFragment } from 'react-relay/hooks';
import { Link as LinkRouter } from 'react-router-dom';
import ListSkeleton from '../skeletons/ListSkeleton';
import LabelList from '../workspace/labels/LabelList';
import NestedNamespaceList from './NestedNamespaceList';
import { NamespaceListItemFragment_namespace$key } from './__generated__/NamespaceListItemFragment_namespace.graphql';
import NamespaceAvatar from '../common/NamespaceAvatar';

interface Props {
    namespaceKey: NamespaceListItemFragment_namespace$key
    last?: boolean
    // Show the path relative to this ancestor instead of the name, e.g. for search results spanning
    // several levels; an empty string shows the full path
    pathRelativeTo?: string
    // Whether a group with children can be expanded in place to show them
    expandable?: boolean
}

function NamespaceListItem({ namespaceKey, last, pathRelativeTo, expandable = true }: Props) {
    const theme = useTheme();

    const data = useFragment<NamespaceListItemFragment_namespace$key>(graphql`
        fragment NamespaceListItemFragment_namespace on Namespace {
            __typename
            id
            name
            description
            fullPath
            ... on Group {
                hasChildren
            }
            ... on Workspace {
                destroyed
                labels {
                    key
                    value
                }
            }
        }
    `, namespaceKey);

    const isGroup = data.__typename === 'Group';
    const label = pathRelativeTo === undefined
        ? data.name
        : pathRelativeTo ? data.fullPath.slice(pathRelativeTo.length + 1) : data.fullPath;
    const hasChildren = expandable && isGroup && data.hasChildren === true;
    const [expanded, setExpanded] = useState(false);

    return (
        <ListItem disablePadding divider={!last} sx={{ flexDirection: 'column', alignItems: 'stretch' }}>
            <Box display="flex" alignItems="center" width="100%">
                <ListItemButton
                    // A group with children expands on row click; only its name navigates
                    {...(hasChildren
                        ? { onClick: () => setExpanded(v => !v), 'aria-expanded': expanded }
                        : { component: LinkRouter, to: `/groups/${data.fullPath}` })}
                    sx={{ flex: 1, minWidth: 0 }}
                >
                    <Tooltip title={isGroup ? 'Group' : 'Workspace'} placement="top">
                        <NamespaceAvatar name={data.name} type={isGroup ? 'group' : 'workspace'} sx={{ marginRight: 2 }} />
                    </Tooltip>
                    <Box sx={{
                        display: 'flex',
                        flexDirection: 'column',
                        minWidth: 0,
                        [theme.breakpoints.down('md')]: {
                            '& > *:nth-of-type(2)': { marginTop: 0.5 }
                        },
                        [theme.breakpoints.up('md')]: {
                            flexGrow: 1,
                            flexDirection: 'row',
                            alignItems: 'center',
                            justifyContent: 'space-between'
                        }
                    }}>
                        <Box sx={{ flex: 1, minWidth: 0 }}>
                            <Stack direction="row" spacing={1} alignItems="center">
                                {hasChildren ? (
                                    <Link
                                        component={LinkRouter}
                                        to={`/groups/${data.fullPath}`}
                                        onClick={(e: React.MouseEvent) => e.stopPropagation()}
                                        underline="hover"
                                        variant="body1"
                                        color="textPrimary"
                                        sx={{ fontWeight: '500' }}
                                    >
                                        {label}
                                    </Link>
                                ) : (
                                    <Link component="div" underline="hover" variant="body1" color="textPrimary" sx={{ fontWeight: '500' }}>
                                        {label}
                                    </Link>
                                )}
                                {!isGroup && data.destroyed && (
                                    <Chip size="small" label="Destroyed" sx={{ color: 'runStatus.destroy' }} />
                                )}
                            </Stack>
                            <Typography variant="body2" color="textSecondary">{data.description}</Typography>
                            {!isGroup && data.labels && data.labels.length > 0 && (
                                <Box sx={{ mt: 0.5 }}>
                                    <LabelList labels={data.labels} size="xs" maxVisible={3} />
                                </Box>
                            )}
                        </Box>
                    </Box>
                    {hasChildren && (
                        // Inside the row button so it shares the hover highlight; the row itself toggles
                        <ExpandMoreIcon color="action" aria-hidden="true" sx={{
                            ml: 1,
                            flexShrink: 0,
                            transition: 'transform 0.2s',
                            transform: expanded ? 'rotate(180deg)' : 'none'
                        }} />
                    )}
                </ListItemButton>
            </Box>
            {expanded && hasChildren && (
                <Box sx={{ pl: 4, borderTop: `1px solid ${theme.palette.divider}` }}>
                    <Suspense fallback={<ListSkeleton rowCount={3} />}>
                        <NestedNamespaceList parentId={data.id} />
                    </Suspense>
                </Box>
            )}
        </ListItem>
    );
}

export default NamespaceListItem;
