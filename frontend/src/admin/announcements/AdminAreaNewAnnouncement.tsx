import { useState, useMemo } from "react";
import graphql from "babel-plugin-relay/macro";
import { Box, Button, Divider, Typography } from "@mui/material";
import LoadingButton from '@mui/lab/LoadingButton';
import { Link as RouterLink, useNavigate } from "react-router-dom";
import { MutationError } from "../../common/error";
import AdminAreaAnnouncementForm, { FormData } from "./AdminAreaAnnouncementForm";
import { useMutation } from "react-relay/hooks";
import AdminAreaBreadcrumbs from "../AdminAreaBreadcrumbs";
import { AdminAreaNewAnnouncementMutation } from "./__generated__/AdminAreaNewAnnouncementMutation.graphql";

function AdminAreaNewAnnouncement() {
    const navigate = useNavigate();
    const [error, setError] = useState<MutationError>();

    const [formData, setFormData] = useState<FormData>({
        message: '',
        type: 'INFO',
        dismissible: true,
        startTime: null,
        endTime: null
    });

    const isFormValid = useMemo(() => {
        return formData.message.trim().length > 0;
    }, [formData]);

    const [commit, isInFlight] = useMutation<AdminAreaNewAnnouncementMutation>(graphql`
        mutation AdminAreaNewAnnouncementMutation($input: CreateAnnouncementInput!) {
            createAnnouncement(input: $input) {
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

    const onSave = () => {
        setError(undefined);

        commit({
            variables: {
                input: {
                    message: formData.message,
                    type: formData.type,
                    dismissible: formData.dismissible,
                    startTime: formData.startTime?.toISOString() || null,
                    endTime: formData.endTime?.toISOString() || null
                }
            },
            onCompleted: data => {
                if (data.createAnnouncement.problems.length) {
                    setError({
                        severity: 'warning',
                        message: data.createAnnouncement.problems.map((problem: any) => problem.message).join(', ')
                    });
                } else if (!data.createAnnouncement.announcement) {
                    setError({
                        severity: 'error',
                        message: 'Unexpected error occurred'
                    });
                } else {
                    navigate('/admin/announcements');
                }
            },
            onError: (error) => {
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
                    { title: "new", path: 'new' }
                ]}
            />
            <Typography variant="h5">New Announcement</Typography>
            <AdminAreaAnnouncementForm
                data={formData}
                onChange={(data: FormData) => setFormData(data)}
                error={error}
            />
            <Divider sx={{ opacity: 0.6 }} />
            <Box marginTop={4}>
                <LoadingButton
                    sx={{ mr: 2 }}
                    loading={isInFlight}
                    disabled={!isFormValid}
                    variant="outlined"
                    color="primary"
                    onClick={onSave}
                >
                    Create Announcement
                </LoadingButton>
                <Button component={RouterLink} color="inherit" to={-1 as any}>
                    Cancel
                </Button>
            </Box>
        </Box>
    );
}

export default AdminAreaNewAnnouncement;
