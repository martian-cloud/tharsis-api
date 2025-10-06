import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useFragment } from 'react-relay/hooks';
import { TerraformIcon } from '../../common/Icons';
import ActivityEventLink from '../ActivityEventLink';
import ActivityEventListItem from '../ActivityEventListItem';
import { ActivityEventTerraformModuleVersionTargetFragment_event$key } from './__generated__/ActivityEventTerraformModuleVersionTargetFragment_event.graphql';

const ACTION_TEXT = {
    CREATE: 'created for',
} as any;

interface Props {
    fragmentRef: ActivityEventTerraformModuleVersionTargetFragment_event$key
}

function ActivityEventTerraformModuleVersionTarget({ fragmentRef }: Props) {
    const data = useFragment<ActivityEventTerraformModuleVersionTargetFragment_event$key>(
        graphql`
        fragment ActivityEventTerraformModuleVersionTargetFragment_event on ActivityEvent
        {
            action
            namespacePath
            target {
                ...on TerraformModuleVersion {
                    version
                    module {
                        name
                        system
                        registryNamespace
                    }
                }
            }
            ...ActivityEventListItemFragment_event
        }
      `, fragmentRef);

    const actionText = ACTION_TEXT[data.action];
    const terraformModuleVersion = data.target as any;

    return (
        <ActivityEventListItem
            fragmentRef={data}
            icon={<TerraformIcon />}
            primary={<React.Fragment>
                Terraform module version <ActivityEventLink to={`/module-registry/${terraformModuleVersion.module?.registryNamespace}/${terraformModuleVersion.module?.name}/${terraformModuleVersion.module?.system}/${terraformModuleVersion.version}`}>{terraformModuleVersion.version}</ActivityEventLink> {actionText} {terraformModuleVersion.module?.registryNamespace}/{terraformModuleVersion.module?.name}/{terraformModuleVersion.module?.system}
            </React.Fragment>}
        />
    );
}

export default ActivityEventTerraformModuleVersionTarget;
