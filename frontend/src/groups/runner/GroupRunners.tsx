import graphql from 'babel-plugin-relay/macro';
import { useFragment } from "react-relay";
import { Route, Routes } from 'react-router-dom';
import EditGroupRunner from './EditGroupRunner';
import GroupRunnerDetails from './GroupRunnerDetails';
import GroupRunnersList from './GroupRunnersList';
import NewGroupRunner from './NewGroupRunner';
import { GroupRunnersFragment_group$key } from './__generated__/GroupRunnersFragment_group.graphql';

interface Props {
    fragmentRef: GroupRunnersFragment_group$key
}

function GroupRunners({ fragmentRef }: Props) {
    const group = useFragment<GroupRunnersFragment_group$key>(
        graphql`
            fragment GroupRunnersFragment_group on Group {
                ...GroupRunnersListFragment_group
                ...NewGroupRunnerFragment_group
                ...EditGroupRunnerFragment_group
                ...GroupRunnerDetailsFragment_group
            }
        `,
        fragmentRef
    );

    return (
        <Routes>
            <Route index element={<GroupRunnersList fragmentRef={group} />} />
            <Route path="new" element={<NewGroupRunner fragmentRef={group} />} />
            <Route path={`:runnerId/edit`} element={<EditGroupRunner fragmentRef={group} />} />
            <Route path={`:runnerId`} element={<GroupRunnerDetails fragmentRef={group} />} />
        </Routes>
    );
}

export default GroupRunners;
