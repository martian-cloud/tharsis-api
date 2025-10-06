import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { TerraformIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventTerraformProviderTargetFragment_event$key } from './__generated__/ActivityEventTerraformProviderTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventTerraformProviderTargetFragment_event$key
}

function ActivityEventTerraformProviderTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventTerraformProviderTargetFragment_event$key>(
        graphql`
        fragment ActivityEventTerraformProviderTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on TerraformProvider {
                    name
                    registryNamespace
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const terraformProvider = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<TerraformIcon />}
            primary={<React.Fragment>
                Terraform provider <ActivityEventLink to={`/provider-registry/${terraformProvider.registryNamespace}/${terraformProvider.name}`}>{terraformProvider.name}</ActivityEventLink> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventTerraformProviderTarget;
