import { Paper, ToggleButton, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useEffect, useMemo, useState } from 'react';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import SearchInput from '../../../common/SearchInput';
import { RunDetailsPlanDiffViewerFragment_run$key, TerraformResourceMode } from './__generated__/RunDetailsPlanDiffViewerFragment_run.graphql';
import { RunDetailsPlanDiffViewerQuery } from './__generated__/RunDetailsPlanDiffViewerQuery.graphql';
import RunDetailsPlanDiffPanel, { Props as PlanDiffProps } from './RunDetailsPlanDiffPanel';

export const MaxDiffSize = 1024 * 1024; // 1MB

interface Props {
    runId: string
}

function RunDetailsPlanDiffViewerContainer({ runId }: Props) {
    const queryData = useLazyLoadQuery<RunDetailsPlanDiffViewerQuery>(graphql`
        query RunDetailsPlanDiffViewerQuery($id: String!) {
            node(id: $id) {
                ... on Run {
                    id
                    ...RunDetailsPlanDiffViewerFragment_run
                }
            }
        }
    `, { id: runId }, { fetchPolicy: 'store-and-network' });

    if (!queryData.node) {
        return null;
    }

    return <RunDetailsPlanDiffViewer fragmentRef={queryData.node} />;
}

function getResourceKey(resource: any) {
    const mode = resource.mode as TerraformResourceMode;
    return mode === 'managed' ? `resource / ${resource.address}` : `data / ${resource.address}`;
}

function getOutputKey(output: any) {
    return `output / ${output.outputName}`;
}

function RunDetailsPlanDiffViewer({ fragmentRef }: { fragmentRef: RunDetailsPlanDiffViewerFragment_run$key }) {
    const data = useFragment<RunDetailsPlanDiffViewerFragment_run$key>(
        graphql`
        fragment RunDetailsPlanDiffViewerFragment_run on Run
        {
            plan {
                status
                changes {
                    resources {
                        action
                        address
                        providerName
                        resourceType
                        resourceName
                        moduleAddress
                        mode
                        unifiedDiff
                        originalSource
                        drifted
                        imported
                        warnings {
                            line
                            message
                            changeType
                        }
                    }
                    outputs {
                        action
                        outputName
                        unifiedDiff
                        originalSource
                        warnings {
                            line
                            message
                            changeType
                        }
                    }
                }
            }
        }
      `, fragmentRef);

    const [collapseAll, setCollapseAll] = useState<boolean>(false);
    const [collapsedState, setCollapsedState] = useState<{ [key: string]: boolean }>({});
    const [search, setSearch] = useState<string>('');

    useEffect(() => {
        Object.values(collapsedState).some((collapsed) => !collapsed) && setCollapseAll(false);
    }, [collapsedState]);

    const onCollapseAllChange = (collapsed: boolean) => {
        const newCollapsedState: { [key: string]: boolean } = {};
        data.plan.changes?.resources.forEach((resource) => {
            newCollapsedState[getResourceKey(resource)] = collapsed;
        });
        data.plan.changes?.outputs.forEach((output) => {
            newCollapsedState[getOutputKey(output)] = collapsed;
        });
        setCollapsedState(newCollapsedState);
        setCollapseAll(collapsed);
    };

    const throttledSetSearch = useMemo(
        () =>
            throttle(
                (input: string) => {
                    setSearch(input);
                },
                1000,
                { leading: false, trailing: true }
            ),
        [setSearch],
    );

    const onKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        // Only handle enter key type
        if (event.which === 13) {
            throttledSetSearch.flush();
        }
    };

    const filteredDiffs = useMemo(() => {
        const response: PlanDiffProps[] = [];

        // Build resource diffs
        data.plan.changes?.resources.map((resource) => {
            const key = getResourceKey(resource);
            response.push({
                title: key,
                action: resource.action,
                drift: resource.drifted,
                imported: resource.imported,
                diff: resource.unifiedDiff,
                oldSrc: resource.originalSource,
                warnings: resource.warnings,
                collapsed: !!collapsedState[key],
                onCollapseChange: (collapsed) => setCollapsedState({ ...collapsedState, [key]: collapsed })
            });
        });

        // Build output diffs
        data.plan.changes?.outputs.map((output) => {
            const key = getOutputKey(output);
            response.push({
                title: key,
                action: output.action,
                drift: false,
                imported: false,
                diff: output.unifiedDiff,
                oldSrc: output.originalSource,
                warnings: output.warnings,
                collapsed: !!collapsedState[key],
                onCollapseChange: (collapsed) => setCollapsedState({ ...collapsedState, [key]: collapsed })
            });
        });

        // Filter diffs based on search
        const normalizedSearch = search.toLowerCase();
        return normalizedSearch !== '' ? response.filter((diff) => diff.title.toLowerCase().includes(normalizedSearch)) : response;
    }, [data.plan.changes, search, collapsedState]);

    return (
        <React.Fragment>
            {filteredDiffs.length === 0 && search === '' && <Paper
                variant="outlined"
                sx={{
                    minHeight: 100,
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'center',
                }}
            >
                <Typography color="textSecondary" align="center">
                    This plan does not contain any changes
                </Typography>
            </Paper>}
            {(filteredDiffs.length > 0 || search !== '') && <Box>
                <Box mb={2} display="flex" alignItems="center">
                    <SearchInput
                        fullWidth
                        placeholder="search for change by resource or output name"
                        onChange={(event: React.ChangeEvent<HTMLInputElement>) => throttledSetSearch(event.target.value)}
                        onKeyPress={onKeyPress}
                    />
                    <Box ml={2}>
                        <ToggleButton
                            sx={{ whiteSpace: 'nowrap' }}
                            onChange={() => onCollapseAllChange(!collapseAll)}
                            color="secondary"
                            selected={collapseAll}
                            size="small"
                            value="collapse">
                            Collapse All
                        </ToggleButton>
                    </Box>
                </Box>
                {filteredDiffs.length === 0 && <Box minHeight={100} display="flex" flexDirection="column" justifyContent="center">
                    <Typography color="textSecondary" align="center">No changes matching search <strong>{search}</strong></Typography>
                </Box>}
                {filteredDiffs.map((diff) => {
                    return (
                        <RunDetailsPlanDiffPanel
                            key={diff.title}
                            title={diff.title}
                            action={diff.action}
                            drift={diff.drift}
                            imported={diff.imported}
                            diff={diff.diff}
                            oldSrc={diff.oldSrc}
                            warnings={diff.warnings}
                            collapsed={diff.collapsed}
                            onCollapseChange={diff.onCollapseChange}
                        />
                    );
                })}
            </Box>}
        </React.Fragment>
    );
}

export default RunDetailsPlanDiffViewerContainer;
