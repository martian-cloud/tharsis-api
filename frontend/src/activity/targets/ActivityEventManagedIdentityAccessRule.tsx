import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { ManagedIdentityIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventManagedIdentityAccessRuleTargetFragment_event$key } from './__generated__/ActivityEventManagedIdentityAccessRuleTargetFragment_event.graphql';

const ACTION_TEXT = {
    UPDATE: 'updated',
    CREATE: 'created',
} as any;

interface Props {
    fragmentRef: ActivityEventManagedIdentityAccessRuleTargetFragment_event$key
}

function ActivityEventManagedIdentityAccessRuleTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventManagedIdentityAccessRuleTargetFragment_event$key>(
        graphql`
        fragment ActivityEventManagedIdentityAccessRuleTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                __typename
                ...on ManagedIdentityAccessRule {
                    runStage
                    managedIdentity {
                        id
                        resourcePath
                    }
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const rule = data.target as any;

    const managedIdentityLink = <ActivityEventLink to={`/groups/${data.namespacePath}/-/managed_identities/${rule.managedIdentity?.id}`}>{rule.managedIdentity?.resourcePath}</ActivityEventLink>

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<ManagedIdentityIcon />}
            primary={<React.Fragment>
                Managed identity {managedIdentityLink} access rule {actionText} for the <Typography component="span" sx={{ fontWeight: 500 }}>{rule.runStage}</Typography> stage
            </React.Fragment>}
        />
    );
}

export default ActivityEventManagedIdentityAccessRuleTarget;
