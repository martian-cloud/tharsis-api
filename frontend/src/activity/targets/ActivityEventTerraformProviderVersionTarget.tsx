import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { TerraformIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventTerraformProviderVersionTargetFragment_event$key } from './__generated__/ActivityEventTerraformProviderVersionTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created for',
} as any;

interface Props {
    fragmentRef: ActivityEventTerraformProviderVersionTargetFragment_event$key
}

function ActivityEventTerraformProviderVersionTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventTerraformProviderVersionTargetFragment_event$key>(
        graphql`
        fragment ActivityEventTerraformProviderVersionTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on TerraformProviderVersion {
                    version
                    provider {
                        name
                        registryNamespace
                    }
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const terraformProviderVersion = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<TerraformIcon />}
            primary={<React.Fragment>
                Terraform provider version <ActivityEventLink to={`/provider-registry/${terraformProviderVersion.provider?.registryNamespace}/${terraformProviderVersion.provider?.name}/${terraformProviderVersion.version}`}>{terraformProviderVersion.version}</ActivityEventLink> {actionText} {terraformProviderVersion.provider?.registryNamespace}/{terraformProviderVersion.provider?.name}
            </React.Fragment>}
        />
    );
}

export default ActivityEventTerraformProviderVersionTarget;
