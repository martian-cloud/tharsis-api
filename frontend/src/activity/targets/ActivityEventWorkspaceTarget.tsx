import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { WorkspaceIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventWorkspaceTargetFragment_event$key } from './__generated__/ActivityEventWorkspaceTargetFragment_event.graphql';

const ACTION_TEXT = {
    MIGRATE: 'migrated from',
    CREATE_MEMBERSHIP: 'added to',
    CREATE: 'created',
    DELETE_CHILD_RESOURCE: 'deleted',
    LOCK: 'locked',
    REMOVE_MEMBERSHIP: 'removed from',
    SET_VARIABLES: 'variables updated',
    UNLOCK: 'unlocked',
    UPDATE: 'updated',
} as any;

const RESOURCE_TYPES = {
    VARIABLE: 'Variable',
} as any;

const MEMBER_TYPES = {
    User: 'User',
    ServiceAccount: 'Service account',
    Team: 'Team'
} as any;

function getMemberIdentifier(member: any) {
    if (!member) {
        return 'n/a';
    }
    switch (member.__typename) {
        case 'User':
            return member.username;
        case 'ServiceAccount':
            return member.resourcePath;
        case 'Team':
            return member.name;
    }
}

interface Props {
    fragmentRef: ActivityEventWorkspaceTargetFragment_event$key
}

function ActivityEventWorkspaceTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventWorkspaceTargetFragment_event$key>(
        graphql`
        fragment ActivityEventWorkspaceTargetFragment_event on ActivityEvent
        {
            action
            target {
                ...on Workspace {
                    name
                    fullPath
                    description
                }
            }
            payload {
                __typename
                ...on ActivityEventDeleteChildResourcePayload {
                    name
                    type
                }
                ...on ActivityEventCreateNamespaceMembershipPayload {
                    member {
                      __typename
                      ... on User {
                        username
                      }
                      ... on ServiceAccount {
                        resourcePath
                      }
                      ... on Team {
                        name
                      }
                    }
                    role
                  }
                ...on ActivityEventRemoveNamespaceMembershipPayload {
                    member {
                      __typename
                      ... on User {
                        username
                      }
                      ... on ServiceAccount {
                        resourcePath
                      }
                      ... on Team {
                        name
                      }
                    }
                }
                ...on ActivityEventMigrateWorkspacePayload {
                    previousGroupPath
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const workspace = data.target as any;
    const payload = data.payload as any;

    const namespaceLink = <ActivityEventLink to={`/groups/${workspace.fullPath}`}>{workspace.name}</ActivityEventLink>;

    let primary;

    if (['CREATE', 'UPDATE', 'LOCK', 'UNLOCK', 'SET_VARIABLES'].includes(data.action)) {
        primary = <React.Fragment>Workspace {namespaceLink} {actionText}</React.Fragment>;
    } else if ('CREATE_MEMBERSHIP' === data.action) {
        primary = <React.Fragment>{MEMBER_TYPES[payload?.member?.__typename] || 'Unknown member type'} <Typography component="span" sx={{ fontWeight: 500 }}>{getMemberIdentifier(payload?.member)}</Typography> added to workspace {namespaceLink} with role {payload?.role}</React.Fragment>;
    } else if ('REMOVE_MEMBERSHIP' === data.action) {
        primary = <React.Fragment>{MEMBER_TYPES[payload?.member?.__typename] || 'Unknown member type'} <Typography component="span" sx={{ fontWeight: 500 }}>{getMemberIdentifier(payload?.member)}</Typography> removed from workspace {namespaceLink}</React.Fragment>;
    } else if (data.action === 'DELETE_CHILD_RESOURCE') {
        primary = <React.Fragment>{RESOURCE_TYPES[payload.type] || 'Unknown resource type'} with name <Typography component="span" sx={{ fontWeight: 500 }}>{payload.name}</Typography> deleted from workspace {namespaceLink}</React.Fragment>;
    } else if ('MIGRATE' === data.action) {
        primary = <React.Fragment>Workspace {namespaceLink} {actionText} <Typography component="span" sx={{ fontWeight: 500 }}>{payload?.previousGroupPath}</Typography></React.Fragment>;
    }

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<WorkspaceIcon />}
            primary={primary}
        />
    );
}

export default ActivityEventWorkspaceTarget;
