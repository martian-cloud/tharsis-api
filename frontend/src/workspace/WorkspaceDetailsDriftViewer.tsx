import { ToggleButton, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import throttle from 'lodash.throttle';
import React, { useEffect, useMemo, useState } from 'react';
import { useFragment, useLazyLoadQuery } from 'react-relay/hooks';
import SearchInput from '../common/SearchInput';
import { WorkspaceDetailsDriftViewerFragment_workspace$key, TerraformResourceMode } from './__generated__/WorkspaceDetailsDriftViewerFragment_workspace.graphql';
import { WorkspaceDetailsDriftViewerQuery } from './__generated__/WorkspaceDetailsDriftViewerQuery.graphql';
import RunDetailsPlanDiffPanel, { Props as DiffProps } from './runs/plandiff/RunDetailsPlanDiffPanel';

interface Props {
    workspaceId: string
}

function WorkspaceDetailsDriftViewerContainer({ workspaceId }: Props) {
    const queryData = useLazyLoadQuery<WorkspaceDetailsDriftViewerQuery>(graphql`
        query WorkspaceDetailsDriftViewerQuery($id: String!) {
            node(id: $id) {
                ... on Workspace {
                    id
                    ...WorkspaceDetailsDriftViewerFragment_workspace
                }
            }
        }
    `, { id: workspaceId }, { fetchPolicy: 'store-and-network' });

    if (!queryData.node) {
        return null;
    }

    return <WorkspaceDetailsDriftViewer fragmentRef={queryData.node} />;
}

function getResourceKey(resource: any) {
    const mode = resource.mode as TerraformResourceMode;
    return mode === 'managed' ? `resource / ${resource.address}` : `data / ${resource.address}`;
}

function WorkspaceDetailsDriftViewer({ fragmentRef }: { fragmentRef: WorkspaceDetailsDriftViewerFragment_workspace$key }) {
    const data = useFragment<WorkspaceDetailsDriftViewerFragment_workspace$key>(
        graphql`
        fragment WorkspaceDetailsDriftViewerFragment_workspace on Workspace
        {
            assessment {
                hasDrift
                startedAt
                completedAt
                run {
                    status
                    plan {
                        changes {
                            resources {
                                action
                                originalSource
                                mode
                                address
                                imported
                                drifted
                                imported
                                unifiedDiff
                                warnings {
                                    line
                                    message
                                    changeType
                                }
                            }
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
        data.assessment?.run?.plan.changes?.resources.forEach((resource) => {
            newCollapsedState[getResourceKey(resource)] = collapsed;
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
        const response: DiffProps[] = [];

        // Build resource diffs
        data.assessment?.run?.plan.changes?.resources
            .filter(resource => resource.drifted) // Only include resources with drift
            .map((resource) => {
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

        //Filter diffs based on search
        const normalizedSearch = search.toLowerCase();
        return normalizedSearch !== '' ? response.filter((diff) => diff.title.toLowerCase().includes(normalizedSearch)) : response;
    }, [data.assessment?.run?.plan.changes, search, collapsedState]);

    return (
        <React.Fragment>
            {(filteredDiffs.length > 0 || search !== '') && <Box>
                <Box mb={2} display="flex" alignItems="center">
                    <SearchInput
                        fullWidth
                        placeholder="search by resource name"
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
        </React.Fragment>
    );
}

export default WorkspaceDetailsDriftViewerContainer;
