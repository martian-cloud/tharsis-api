import { useState, useEffect, useCallback } from 'react';
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, ListItemButton, ListItemText } from '@mui/material';
import { useNavigate } from 'react-router-dom';
import { useMutation } from 'react-relay/hooks';
import graphql from 'babel-plugin-relay/macro';
import humanizeDuration from 'humanize-duration';

interface Props {
    expiresAt: string;
    onMenuClose: () => void;
}

function DeactivateAdminModeListItem({ expiresAt, onMenuClose }: Props) {
    const [display, setDisplay] = useState('');
    const [showConfirm, setShowConfirm] = useState(false);
    const navigate = useNavigate();

    const [commitDeactivate] = useMutation(graphql`
        mutation DeactivateAdminModeListItemMutation($input: DeactivateAdminModeInput!) {
            deactivateAdminMode(input: $input) {
                user {
                    id
                    adminModeEnabled
                    adminModeExpiration
                }
                problems {
                    message
                    type
                    field
                }
            }
        }
    `);

    const handleDeactivate = useCallback(() => {
        commitDeactivate({
            variables: { input: {} },
            updater: (store) => {
                store.invalidateStore();
            },
            onCompleted: () => {
                navigate('/');
            },
        });
    }, [commitDeactivate, navigate]);

    useEffect(() => {
        const update = () => {
            const diff = new Date(expiresAt).getTime() - Date.now();
            if (diff <= 0) {
                handleDeactivate();
                return;
            }
            setDisplay(humanizeDuration(diff, { round: true, largest: 2 }));
        };
        update();
        const interval = setInterval(update, 1000);
        return () => clearInterval(interval);
    }, [expiresAt, handleDeactivate]);

    return (
        <>
            <ListItemButton onClick={() => setShowConfirm(true)}>
                <ListItemText primary={`Deactivate Admin Mode (${display})`} />
            </ListItemButton>
            <Dialog open={showConfirm} onClose={() => setShowConfirm(false)} maxWidth="xs" fullWidth>
                <DialogTitle>Deactivate Admin Mode</DialogTitle>
                <DialogContent>
                    Are you sure you want to deactivate admin mode?
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setShowConfirm(false)}>Cancel</Button>
                    <Button onClick={() => { setShowConfirm(false); onMenuClose(); handleDeactivate(); }} variant="contained" color="error">Deactivate</Button>
                </DialogActions>
            </Dialog>
        </>
    );
}

export default DeactivateAdminModeListItem;
