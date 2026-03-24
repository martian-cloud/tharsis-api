import { Box, Chip, Tooltip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { TerraformIcon } from '../../common/Icons';
import LabelList from '../../workspace/labels/LabelList';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventTerraformModuleTargetFragment_event$key } from './__generated__/ActivityEventTerraformModuleTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventTerraformModuleTargetFragment_event$key
}

function ActivityEventTerraformModuleTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventTerraformModuleTargetFragment_event$key>(
        graphql`
        fragment ActivityEventTerraformModuleTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on TerraformModule {
                    name
                    system
                    registryNamespace
                }
            }
            payload {
                __typename
                ...on ActivityEventCreateTerraformModulePayload {
                    labels {
                        key
                        value
                    }
                }
                ...on ActivityEventUpdateTerraformModulePayload {
                    labelChanges {
                        added {
                            key
                            value
                        }
                        updated {
                            key
                            value
                        }
                        removed
                    }
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const terraformModule = data.target as any;
    const payload = data.payload as any;

    const moduleLink = <ActivityEventLink to={`/module-registry/${terraformModule.registryNamespace}/${terraformModule.name}/${terraformModule.system}`}>{terraformModule.registryNamespace}/{terraformModule.name}/{terraformModule.system}</ActivityEventLink>;
    const groupLink = <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>;

    const hasLabelChanges = data.action === 'UPDATE' &&
        payload?.__typename === 'ActivityEventUpdateTerraformModulePayload' &&
        payload.labelChanges &&
        ((payload.labelChanges.added?.length > 0) ||
            (payload.labelChanges.updated?.length > 0) ||
            (payload.labelChanges.removed?.length > 0));

    const hasCreateLabels = data.action === 'CREATE' &&
        payload?.__typename === 'ActivityEventCreateTerraformModulePayload' &&
        payload.labels?.length > 0;

    let primaryVerb: React.ReactNode = actionText;
    if (hasLabelChanges) {
        primaryVerb = 'labels updated';
    } else if (hasCreateLabels) {
        primaryVerb = 'created with labels';
    }

    const primary = <React.Fragment>
        Terraform module {moduleLink} {primaryVerb} in group {groupLink}
    </React.Fragment>;

    let secondary;

    if (hasCreateLabels) {
        secondary = (
            <LabelList labels={payload.labels as any} prefix="Labels" size="small" maxVisible={5} />
        );
    } else if (hasLabelChanges) {
        secondary = (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                {payload.labelChanges.added?.length > 0 && (
                    <LabelList labels={payload.labelChanges.added as any} prefix="Added" size="small" maxVisible={5} />
                )}
                {payload.labelChanges.updated?.length > 0 && (
                    <LabelList labels={payload.labelChanges.updated as any} prefix="Updated" size="small" maxVisible={5} />
                )}
                {payload.labelChanges.removed?.length > 0 && (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, flexWrap: 'wrap' }}>
                        <Typography variant="body2" color="textSecondary" component="span">
                            Removed:
                        </Typography>
                        {payload.labelChanges.removed.map((key: string, index: number) => (
                            <Tooltip key={index} title={key} arrow>
                                <Chip
                                    label={key}
                                    size="small"
                                    variant="outlined"
                                    sx={{
                                        height: 20,
                                        fontSize: '0.75rem',
                                        '& .MuiChip-label': {
                                            px: 1,
                                            py: 0,
                                            textDecoration: 'line-through'
                                        }
                                    }}
                                />
                            </Tooltip>
                        ))}
                    </Box>
                )}
            </Box>
        );
    }

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<TerraformIcon />}
            primary={primary}
            secondary={secondary}
        />
    );
}

export default ActivityEventTerraformModuleTarget;
