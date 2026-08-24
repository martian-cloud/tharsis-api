import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import { Route, Routes } from 'react-router-dom';
import EditPackageVersion from './EditPackageVersion';
import NewPackageVersion from './NewPackageVersion';
import GroupEditPackage from './GroupEditPackage';
import GroupPackageVersionDetails from './GroupPackageVersionDetails';
import GroupNewPackage from './GroupNewPackage';
import GroupPackageList from './GroupPackageList';
import { GroupPackagesFragment_group$key } from './__generated__/GroupPackagesFragment_group.graphql';

interface Props {
    fragmentRef: GroupPackagesFragment_group$key
}

function GroupPackages(props: Props) {
    const data = useFragment<GroupPackagesFragment_group$key>(
        graphql`
        fragment GroupPackagesFragment_group on Group
        {
            ...GroupPackageListFragment_group
            ...GroupNewPackageFragment_group
            ...GroupEditPackageFragment_group
        }
      `, props.fragmentRef);

    return (
        <Box>
            <Routes>
                <Route index element={<GroupPackageList fragmentRef={data} />} />
                <Route path="new" element={<GroupNewPackage fragmentRef={data} />} />
                <Route path=":id/edit" element={<GroupEditPackage fragmentRef={data} />} />
                <Route path=":packageId" element={<GroupPackageVersionDetails />} />
                <Route path=":packageId/versions/new" element={<NewPackageVersion />} />
                <Route path=":packageId/versions/:versionId/edit" element={<EditPackageVersion />} />
            </Routes>
        </Box>
    );
}

export default GroupPackages;
