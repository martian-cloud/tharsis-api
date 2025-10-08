import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { TerraformIcon } from '../../common/Icons';
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
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const terraformModule = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<TerraformIcon />}
            primary={<React.Fragment>
                Terraform module <ActivityEventLink to={`/module-registry/${terraformModule.registryNamespace}/${terraformModule.name}/${terraformModule.system}`}>{terraformModule.registryNamespace}/{terraformModule.name}/{terraformModule.system}</ActivityEventLink> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventTerraformModuleTarget;
