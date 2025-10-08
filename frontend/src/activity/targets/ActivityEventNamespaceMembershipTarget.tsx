import { Paper, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { MemberIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventNamespaceMembershipTargetFragment_event$key } from './__generated__/ActivityEventNamespaceMembershipTargetFragment_event.graphql';

const ACTION_TEXT = {
    UPDATE: 'updated',
} as any;

function getMemberIdentifier(member: any) {
    if (!member) {
        return 'n/a';
    }
    switch(member.__typename) {
        case 'User':
            return member.username;
        case 'ServiceAccount':
            return member.resourcePath;
        case 'Team':
            return member.name;
    }
}

const MEMBER_TYPES = {
    User: 'user',
    ServiceAccount: 'service account',
    Team: 'team'
} as any;

interface Props {
    fragmentRef: ActivityEventNamespaceMembershipTargetFragment_event$key
}

function ActivityEventNamespaceMembershipTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventNamespaceMembershipTargetFragment_event$key>(
        graphql`
        fragment ActivityEventNamespaceMembershipTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on NamespaceMembership {
                    member {
                        __typename
                        ...on User {
                            username
                        }
                        ...on Team {
                            name
                        }
                        ...on ServiceAccount {
                            resourcePath
                        }
                    }
                }
            }
            payload {
                __typename
                ...on ActivityEventUpdateNamespaceMembershipPayload {
                    prevRole
                    newRole
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const payload = data.payload as any;
    const target = data.target as any;

    const memberIdentifier = getMemberIdentifier(target.member);

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<MemberIcon />}
            primary={<React.Fragment>
                Membership {actionText} for {MEMBER_TYPES[target.member.__typename]} <Typography component="span" sx={{ fontWeight: 500 }}>{memberIdentifier}</Typography> in namespace <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
            secondary={data.payload?.__typename === 'ActivityEventUpdateNamespaceMembershipPayload' ? <Paper sx={{ padding: `4px 8px` }}>
                <Typography variant="body2">
                    {payload.prevRole} role changed to {payload.newRole}
                </Typography>
            </Paper> : null}
        />
    );
}

export default ActivityEventNamespaceMembershipTarget;
