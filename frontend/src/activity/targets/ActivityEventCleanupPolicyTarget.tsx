import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { CleanupPolicyIcon } from '../../common/Icons';
import { CLEANUP_POLICY_KINDS } from '../../namespace/cleanuppolicy/rules';
import { CleanupPolicyKind } from '../../namespace/cleanuppolicy/types';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventCleanupPolicyTargetFragment_event$key } from './__generated__/ActivityEventCleanupPolicyTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventCleanupPolicyTargetFragment_event$key
}

function ActivityEventCleanupPolicyTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventCleanupPolicyTargetFragment_event$key>(
        graphql`
        fragment ActivityEventCleanupPolicyTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on CleanupPolicy {
                    kind
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action] ?? data.action.toLowerCase();
    const kind = data.target?.kind as CleanupPolicyKind | undefined;
    const kindDetails = kind ? CLEANUP_POLICY_KINDS[kind] : undefined;
    const kindLabel = <Typography component="span" sx={{ fontWeight: 500 }}>{kindDetails?.label ?? kind ?? 'unknown kind'}</Typography>;

    // An unrecognized kind has no page to link to, so it is shown as plain text. The route uses the
    // kind's path slug (lowercase), not the enum value.
    const kindTarget = kindDetails
        ? <ActivityEventLink to={`/groups/${data.namespacePath}/-/cleanup_policies?expand=${kind!.toLowerCase()}`}>{kindLabel}</ActivityEventLink>
        : kindLabel;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<CleanupPolicyIcon />}
            primary={<React.Fragment>
                Cleanup policy for {kindTarget} {actionText} in <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventCleanupPolicyTarget;
