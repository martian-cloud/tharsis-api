import { Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { GroupIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventGroupTargetFragment_event$key } from './__generated__/ActivityEventGroupTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE_MEMBERSHIP: 'added to',
    CREATE: 'created',
    MIGRATE: 'migrated from',
    DELETE_CHILD_RESOURCE: 'deleted',
    REMOVE_MEMBERSHIP: 'removed from',
    SET_VARIABLES: 'variables updated',
    UPDATE: 'updated',
} as any;

const RESOURCE_TYPES = {
    WORKSPACE: 'Workspace',
    GROUP: 'Group',
    MANAGED_IDENTITY: 'Managed identity',
    SERVICE_ACCOUNT: 'Service account',
    GPG_KEY: 'GPG key',
    TERRAFORM_MODULE: 'Terraform module',
    TERRAFORM_PROVIDER: 'Terraform provider',
    VARIABLE: 'Variable',
    VCS_PROVIDER: 'VCS Provider',
    MODULE: 'Module',
    RUNNER: 'Runner agent',
    FEDERATED_REGISTRY: 'Federated Registry'
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
    fragmentRef: ActivityEventGroupTargetFragment_event$key
}

function ActivityEventGroupTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventGroupTargetFragment_event$key>(
        graphql`
        fragment ActivityEventGroupTargetFragment_event on ActivityEvent
        {
            action
            target {
                ...on Group {
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
                ...on ActivityEventMigrateGroupPayload {
                    previousGroupPath
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const group = data.target as any;
    const payload = data.payload as any;

    const namespaceLink = <ActivityEventLink to={`/groups/${group.fullPath}`}>{group.name}</ActivityEventLink>;

    let primary;

    if (['CREATE', 'UPDATE', 'SET_VARIABLES'].includes(data.action)) {
        primary = <React.Fragment>Group {namespaceLink} {actionText}</React.Fragment>;
    } else if ('CREATE_MEMBERSHIP' === data.action) {
        primary = <React.Fragment>{MEMBER_TYPES[payload?.member?.__typename] || 'Unknown member type'} <Typography component="span" sx={{ fontWeight: 500 }}>{getMemberIdentifier(payload?.member)}</Typography> added to group {namespaceLink} with role {payload?.role}</React.Fragment>;
    } else if ('REMOVE_MEMBERSHIP' === data.action) {
        primary = <React.Fragment>{MEMBER_TYPES[payload?.member?.__typename] || 'Unknown member type'} <Typography component="span" sx={{ fontWeight: 500 }}>{getMemberIdentifier(payload?.member)}</Typography> removed from group {namespaceLink}</React.Fragment>;
    } else if (data.action === 'DELETE_CHILD_RESOURCE') {
        primary = <React.Fragment>{RESOURCE_TYPES[payload?.type] || 'Unknown resource type'} with name <Typography component="span" sx={{ fontWeight: 500 }}>{payload?.name || 'unknown'}</Typography> deleted from group {namespaceLink}</React.Fragment>;
    } else if ('MIGRATE' === data.action) {
        primary = <React.Fragment>Group {namespaceLink} {actionText} <Typography component="span" sx={{ fontWeight: 500 }}>{payload?.previousGroupPath}</Typography></React.Fragment>;
    }

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<GroupIcon />}
            primary={primary}
        />
    );
}

export default ActivityEventGroupTarget;
