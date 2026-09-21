import { Chip } from '@mui/material';

interface Props {
    active: boolean;
    expired: boolean;
}

function AdminAreaAnnouncementStatusChip({ active, expired }: Props) {
    if (expired) {
        return <Chip label="Expired" color="default" size="small" variant="outlined" />;
    } else if (active) {
        return <Chip label="Active" color="success" size="small" variant="filled" />;
    } else {
        return <Chip label="Scheduled" color="info" size="small" variant="outlined" />;
    }
}

export default AdminAreaAnnouncementStatusChip;
