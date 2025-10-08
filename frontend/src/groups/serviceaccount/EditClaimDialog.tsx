import { LoadingButton } from '@mui/lab';
import { Button, Dialog, DialogActions, DialogContent, DialogTitle, Typography } from '@mui/material';
import Alert from '@mui/material/Alert';
import TextField from '@mui/material/TextField';
import { useEffect, useState } from 'react';

interface Props {
    claim: any;
    error?: string;
    editMode?: boolean;
    onClose: (claim?: any) => void;
}

function EditClaimDialog(props: Props) {
    const { onClose, claim, editMode, error, ...other } = props;

    const [data, setData] = useState<any>({ ...claim });

    useEffect(() => {
        setData({ ...claim });
    }, [claim]);

    const onUpdate = () => {
        onClose(data);
    };

    return (
        <Dialog
            keepMounted={false}
            maxWidth="sm"
            open={!!claim}
            {...other}
        >
            <DialogTitle>
                {editMode ? 'Edit' : 'New'} Claim
                <Typography component="div" variant="caption" color="textSecondary">
                    The name and value of this claim must be present in the token that is used to login to the service account
                </Typography>
            </DialogTitle>
            <DialogContent dividers>
                {error && <Alert sx={{ marginBottom: 2 }} severity={'warning'}>
                    {error}
                </Alert>}
                <TextField
                    size="small"
                    label="Name"
                    fullWidth
                    defaultValue={data.name}
                    onChange={event => setData({ name: event.target.value, value: data.value })}
                />
                <TextField
                    size="small"
                    margin='normal'
                    fullWidth
                    label="Value"
                    defaultValue={data.value}
                    onChange={event => setData({ name: data.name, value: event.target.value })}
                />
            </DialogContent>
            <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Cancel
                </Button>
                <LoadingButton disabled={data.name === '' || data.value === ''} color="primary" onClick={onUpdate}>
                    {editMode ? 'Update' : 'Create'}
                </LoadingButton>
            </DialogActions>
        </Dialog>
    );
}

export default EditClaimDialog;
