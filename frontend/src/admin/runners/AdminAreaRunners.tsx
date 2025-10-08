import { Route, Routes } from 'react-router-dom';
import AdminAreaRunnerDetails from './AdminAreaRunnerDetails';
import AdminAreaRunnersList from './AdminAreaRunnersList';
import EditAdminAreaRunner from './EditAdminAreaRunner';

function AdminAreaRunners() {
    return (
        <Routes>
            <Route index element={<AdminAreaRunnersList />} />
            <Route index path={`:runnerId/edit`} element={<EditAdminAreaRunner />} />
            <Route index path={`:runnerId`} element={<AdminAreaRunnerDetails />} />
        </Routes>
    );
}

export default AdminAreaRunners
