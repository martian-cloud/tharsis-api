import LoadingButton from '@mui/lab/LoadingButton';
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import { useState } from 'react';
import { MutationError } from '../../../common/error';
import ManagedIdentityRuleForm from './ManagedIdentityRuleForm';

interface Props {
    groupPath: string;
    submitInProgress?: boolean;
    error?: MutationError;
    onSubmit: (rule: any) => void;
    onClose: () => void;
}

function NewManagedIdentityRuleDialog(props: Props) {
    const { groupPath, submitInProgress, error, onSubmit, onClose } = props

    const [rule, setRule] = useState<any>({
        runStage: '',
        allowedUsers: [],
        allowedTeams: [],
        allowedServiceAccounts: [],
        moduleAttestationPolicies: []
    });

    return (
        <Dialog
            fullWidth
            maxWidth="md"
            open>
            <DialogTitle>
                New Rule
            </DialogTitle>
            <DialogContent dividers>
                <ManagedIdentityRuleForm
                    groupPath={groupPath}
                    rule={rule}
                    onChange={setRule}
                    error={error}
                />
            </DialogContent>
            <DialogActions>
                <Button size="small" variant="outlined" onClick={onClose} color="inherit">Cancel</Button>
                <LoadingButton
                    disabled={!rule.runStage}
                    loading={submitInProgress}
                    size="small"
                    variant="contained"
                    color="primary"
                    sx={{ marginLeft: 2 }}
                    onClick={() => onSubmit(rule)}>
                    Create Rule
                </LoadingButton>
            </DialogActions>
        </Dialog>
    )
}

export default NewManagedIdentityRuleDialog;
