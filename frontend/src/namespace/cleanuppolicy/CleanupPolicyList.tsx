import { Box, Button, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo, useState } from 'react';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import NamespaceBreadcrumbs from '../NamespaceBreadcrumbs';
import { CleanupPolicyListQuery } from './__generated__/CleanupPolicyListQuery.graphql';
import CleanupPolicyListItem from './CleanupPolicyListItem';
import { CLEANUP_POLICY_KINDS } from './rules';
import { CleanupPolicyKind } from './types';

const DESCRIPTION = 'Automatically delete resources you no longer need. Each cleanup policy covers one resource kind and applies here and to every namespace beneath it, until a descendant defines its own policy for that kind.';

const query = graphql`
    query CleanupPolicyListQuery($namespacePath: String!) {
        namespace(fullPath: $namespacePath) {
            id
            __typename
            fullPath
            effectiveCleanupPolicies {
                kind
                ...CleanupPolicyListItem_fields
            }
        }
    }
`;

interface Props {
    namespacePath: string;
}

function CleanupPolicyList({ namespacePath }: Props) {
    const theme = useTheme();
    const queryData = useLazyLoadQuery<CleanupPolicyListQuery>(
        query,
        { namespacePath },
        { fetchPolicy: 'store-and-network' }
    );

    const namespace = queryData.namespace;
    const isWorkspace = namespace?.__typename === 'Workspace';

    const kinds = useMemo(
        () => (Object.keys(CLEANUP_POLICY_KINDS) as CleanupPolicyKind[])
            .filter(k => !isWorkspace || !CLEANUP_POLICY_KINDS[k].groupOnly),
        [isWorkspace],
    );

    const [searchParams] = useSearchParams();
    // ?expand=runs from activity event links — that kind starts expanded, all others collapsed.
    const expandSlug = searchParams.get('expand');

    const [collapsed, setCollapsed] = useState<Record<string, boolean>>(
        () => Object.fromEntries(
            (Object.keys(CLEANUP_POLICY_KINDS) as CleanupPolicyKind[]).map(k => [
                k,
                expandSlug ? k.toLowerCase() !== expandSlug : false,
            ])
        )
    );
    const toggle = (k: string) => setCollapsed(prev => ({ ...prev, [k]: !prev[k] }));

    const effectivePolicies = namespace?.effectiveCleanupPolicies ?? ([] as any[]);

    // Only render an item for kinds that have an active policy (local or inherited).
    const activeListKinds = useMemo(
        () => kinds.filter(k => effectivePolicies.find((p: any) => p.kind === k)),
        [kinds, effectivePolicies],
    );

    // Kinds with no active policy at all — these are what the top-level Create handles.
    const hasCreatableKinds = useMemo(
        () => kinds.some(k => !effectivePolicies.find((p: any) => p.kind === k)),
        [kinds, effectivePolicies],
    );

    const hasPolicies = activeListKinds.length > 0;

    return (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={namespacePath}
                childRoutes={[{ title: 'cleanup policies', path: 'cleanup_policies' }]}
            />

            {hasPolicies && (
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    justifyContent: 'space-between',
                    mb: 2,
                    [theme.breakpoints.down('md')]: {
                        flexDirection: 'column',
                        alignItems: 'flex-start',
                        '& > *': { marginBottom: 2 },
                    }
                }}>
                    <Box>
                        <Typography variant="h5" gutterBottom>Cleanup</Typography>
                        <Typography variant="body2">
                            {DESCRIPTION}
                        </Typography>
                    </Box>
                    {hasCreatableKinds && (
                        <Box>
                            <Button sx={{ minWidth: 220 }} component={RouterLink} variant="outlined" to="new">
                                New Cleanup Policy
                            </Button>
                        </Box>
                    )}
                </Box>
            )}

            {hasPolicies ? (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                    {activeListKinds.map(k => {
                        const kindDef = CLEANUP_POLICY_KINDS[k];
                        const activePolicy = effectivePolicies.find((p: any) => p.kind === k);

                        return (
                            <CleanupPolicyListItem
                                key={k}
                                policyRef={activePolicy}
                                namespacePath={namespacePath}
                                label={kindDef.label}
                                description={kindDef.description}
                                expanded={!collapsed[k]}
                                onToggle={() => toggle(k)}
                            />
                        );
                    })}
                </Box>
            ) : (
                <Box sx={{ marginTop: 4 }} display="flex" justifyContent="center">
                    <Box padding={4} display="flex" flexDirection="column" justifyContent="center" alignItems="center" sx={{ maxWidth: 600 }}>
                        <Typography variant="h6">Get started with cleanup policies</Typography>
                        <Typography color="textSecondary" align="center" sx={{ marginBottom: 2 }}>
                            {DESCRIPTION}
                        </Typography>
                        {hasCreatableKinds && (
                            <Button component={RouterLink} variant="outlined" to="new">
                                New Cleanup Policy
                            </Button>
                        )}
                    </Box>
                </Box>
            )}
        </Box>
    );
}

export default CleanupPolicyList;
