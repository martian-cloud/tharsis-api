import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { PackageVariantClosed as PackageIcon } from 'mdi-material-ui';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventPackageTargetFragment_event$key } from './__generated__/ActivityEventPackageTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created',
    UPDATE: 'updated',
} as any;

interface Props {
    fragmentRef: ActivityEventPackageTargetFragment_event$key
}

function ActivityEventPackageTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventPackageTargetFragment_event$key>(
        graphql`
        fragment ActivityEventPackageTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on Package {
                    id
                    name
                    groupPath
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const pkg = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<PackageIcon />}
            primary={<React.Fragment>
                Package <ActivityEventLink to={`/groups/${pkg.groupPath}/-/packages/${pkg.id}`}>{pkg.name}</ActivityEventLink> {actionText} in group <ActivityEventLink to={`/groups/${data.namespacePath}`}>{data.namespacePath}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventPackageTarget;
