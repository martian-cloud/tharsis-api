import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { ContentDuplicate as ProviderMirrorIcon } from 'mdi-material-ui';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventTerraformProviderVersionMirrorTargetFragment_event$key } from './__generated__/ActivityEventTerraformProviderVersionMirrorTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
} as any;

interface Props {
    fragmentRef: ActivityEventTerraformProviderVersionMirrorTargetFragment_event$key
}

function ActivityEventTerraformProviderVersionMirrorTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventTerraformProviderVersionMirrorTargetFragment_event$key>(
        graphql`
        fragment ActivityEventTerraformProviderVersionMirrorTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on TerraformProviderVersionMirror {
                    id
                    version
                    providerAddress
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const mirror = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<ProviderMirrorIcon />}
            primary={<React.Fragment>
                Provider mirror <ActivityEventLink to={`/groups/${data.namespacePath}/-/provider_mirror/${mirror.id}`}>{mirror.providerAddress}/{mirror.version}</ActivityEventLink> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventTerraformProviderVersionMirrorTarget;
