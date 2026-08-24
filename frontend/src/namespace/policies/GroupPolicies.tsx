import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import { GroupPoliciesFragment_group$key } from './__generated__/GroupPoliciesFragment_group.graphql';
import EditPolicy from './EditPolicy';
import NewPolicy from './NewPolicy';
import PolicyDetails from './PolicyDetails';
import PolicyList from './PolicyList';

interface Props {
    fragmentRef: GroupPoliciesFragment_group$key
}

function GroupPolicies(props: Props) {
    const group = useFragment<GroupPoliciesFragment_group$key>(
        graphql`
        fragment GroupPoliciesFragment_group on Group {
            id
            fullPath
        }
        `, props.fragmentRef);

    return (
        <Routes>
            <Route index element={
                <PolicyList
                    ownerId={group.id}
                    ownerPath={group.fullPath}
                />
            } />
            <Route path="new" element={
                <NewPolicy
                    ownerId={group.id}
                    ownerPath={group.fullPath}
                    groupPath={group.fullPath}
                />
            } />
            <Route path=":policyId" element={
                <PolicyDetails
                    ownerId={group.id}
                    ownerPath={group.fullPath}
                    currentGroupPath={group.fullPath}
                />
            } />
            <Route path=":policyId/edit" element={
                <EditPolicy
                    ownerPath={group.fullPath}
                    groupPath={group.fullPath}
                />
            } />
        </Routes>
    );
}

export default GroupPolicies;
