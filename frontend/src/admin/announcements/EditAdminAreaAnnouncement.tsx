import { useState, useMemo } from 'react';
import { Box, Button, Divider, Typography } from '@mui/material';
import LoadingButton from '@mui/lab/LoadingButton';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery, useMutation } from 'react-relay/hooks';
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom';
import moment from 'moment';
import { MutationError } from '../../common/error';
import AdminAreaBreadcrumbs from '../AdminAreaBreadcrumbs';
import AdminAreaAnnouncementForm, { FormData } from './AdminAreaAnnouncementForm';
import { EditAdminAreaAnnouncementQuery } from './__generated__/EditAdminAreaAnnouncementQuery.graphql';
import { EditAdminAreaAnnouncementMutation } from './__generated__/EditAdminAreaAnnouncementMutation.graphql';

function EditAdminAreaAnnouncement() {
    const announcementId = useParams().announcementId as string;
    const navigate = useNavigate();

    const queryData = useLazyLoadQuery<EditAdminAreaAnnouncementQuery>(graphql`
        query EditAdminAreaAnnouncementQuery($id: String!) {
            node(id: $id) {
                ... on Announcement {
                    id
                    message
                    type
                    dismissible
                    startTime
                    endTime
                }
            }
        }
    `, { id: announcementId });

    const [commit, isInFlight] = useMutation<EditAdminAreaAnnouncementMutation>(graphql`
        mutation EditAdminAreaAnnouncementMutation($input: UpdateAnnouncementInput!) {
            updateAnnouncement(input: $input) {
                announcement {
                    id
                    message
                    type
                    dismissible
                    startTime
                    endTime
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const announcement = queryData.node as any;

    const [error, setError] = useState<MutationError>();

    const originalFormData: FormData = useMemo(() => ({
        message: announcement.message,
        type: announcement.type,
        dismissible: announcement.dismissible,
        startTime: announcement.startTime ? moment(announcement.startTime) : null,
        endTime: announcement.endTime ? moment(announcement.endTime) : null
    }), [announcement]);

    const [formData, setFormData] = useState<FormData>(originalFormData);

    const isFormValid = useMemo(() => {
        return formData.message.trim().length > 0;
    }, [formData]);

    const hasFormChanged = useMemo(() => {
        return JSON.stringify(formData) !== JSON.stringify(originalFormData);
    }, [formData, originalFormData]);

    const onUpdate = () => {
        setError(undefined);

        commit({
            variables: {
                input: {
                    id: announcement.id,
                    message: formData.message,
                    type: formData.type,
                    dismissible: formData.dismissible,
                    startTime: formData.startTime?.toISOString() || null,
                    endTime: formData.endTime?.toISOString() || null
                }
            },
            onCompleted: data => {
                if (data.updateAnnouncement.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.updateAnnouncement.problems.map((problem: any) => problem.message).join(', ')
                    });
                } else if (!data.updateAnnouncement.announcement) {
                    setError({
                        severity: 'error',
                        message: 'Unexpected error occurred'
                    });
                } else {
                    navigate('/admin/announcements');
                }
            },
            onError: error => {
                setError({
                    severity: 'error',
                    message: `Unexpected error occurred: ${error.message}`
                });
            }
        });
    };

    return (
        <Box>
            <AdminAreaBreadcrumbs
                childRoutes={[
                    { title: "announcements", path: 'announcements' },
                    { title: `${announcement.id.substring(0, 8)}...`, path: announcement.id },
                    { title: "edit", path: 'edit' }
                ]}
            />
            <Typography variant="h5">Edit Announcement</Typography>
            <AdminAreaAnnouncementForm
                editMode
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider sx={{ opacity: 0.6 }} />
            <Box mt={2}>
                <LoadingButton
                    loading={isInFlight}
                    disabled={!isFormValid || !hasFormChanged}
                    variant="outlined"
                    color="primary"
                    sx={{ mr: 2 }}
                    onClick={onUpdate}
                >
                    Update Announcement
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>
                    Cancel
                </Button>
            </Box>
        </Box>
    );
}

export default EditAdminAreaAnnouncement;
