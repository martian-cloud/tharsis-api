import LoadingButton from '@mui/lab/LoadingButton';
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import { nanoid } from 'nanoid';
import { useEffect, useState } from 'react';
import { MutationError } from '../../../common/error';
import ManagedIdentityRuleForm from './ManagedIdentityRuleForm';

interface Props {
    inputRule: any;
    groupPath: string;
    submitInProgress?: boolean;
    error?: MutationError;
    onSubmit: (rule: any) => void;
    onClose: () => void;
}

function EditManagedIdentityRuleDialog(props: Props) {
    const { inputRule, groupPath, submitInProgress, error, onSubmit, onClose } = props

    const [rule, setRule] = useState<any>();

    useEffect(() => {
        // Add _id field to policies in order to provide uniqueness
        setRule({
            ...inputRule,
            moduleAttestationPolicies: inputRule.moduleAttestationPolicies?.map((p: any) => ({ ...p, _id: nanoid() })) || []
        });
    }, [inputRule]);

    return rule ? (
        <Dialog
            fullWidth
            maxWidth="md"
            open>
            <DialogTitle>
                Edit Rule
            </DialogTitle>
            <DialogContent dividers>
                <ManagedIdentityRuleForm
                    editMode
                    groupPath={groupPath}
                    rule={rule}
                    onChange={setRule}
                    error={error}
                />
            </DialogContent>
            <DialogActions>
                <Button size="small" variant="outlined" onClick={onClose} color="inherit">Cancel</Button>
                <LoadingButton
                    loading={submitInProgress}
                    size="small"
                    variant="contained"
                    color="primary"
                    sx={{ marginLeft: 2 }}
                    onClick={() => onSubmit(rule)}>
                    Update Rule
                </LoadingButton>
            </DialogActions>
        </Dialog>
    ) : null;
}

export default EditManagedIdentityRuleDialog;
