import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { PackageVariantClosed as PackageIcon } from 'mdi-material-ui';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventPackageVersionTargetFragment_event$key } from './__generated__/ActivityEventPackageVersionTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created for',
    UPDATE: 'updated for',
} as any;

interface Props {
    fragmentRef: ActivityEventPackageVersionTargetFragment_event$key
}

function ActivityEventPackageVersionTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventPackageVersionTargetFragment_event$key>(
        graphql`
        fragment ActivityEventPackageVersionTargetFragment_event on ActivityEvent
        {
            action
            target {
                ...on PackageVersion {
                    id
                    version
                    package {
                        id
                        name
                        groupPath
                    }
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const packageVersion = data.target as any;
    const pkg = packageVersion.package;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<PackageIcon />}
            primary={<React.Fragment>
                Package version <ActivityEventLink to={`/groups/${pkg?.groupPath}/-/packages/${pkg?.id}?tab=versions`}>{packageVersion.version}</ActivityEventLink> {actionText} <ActivityEventLink to={`/groups/${pkg?.groupPath}/-/packages/${pkg?.id}`}>{pkg?.name}</ActivityEventLink>
            </React.Fragment>}
        />
    );
}

export default ActivityEventPackageVersionTarget;
