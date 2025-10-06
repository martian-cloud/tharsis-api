import { Route, Routes } from 'react-router-dom';
import AdminAreaUsersList from './AdminAreaUsersList';

function AdminAreaUsers() {
    return (
        <Routes>
            <Route index element={<AdminAreaUsersList />} />
        </Routes>
    );
}

export default AdminAreaUsers
