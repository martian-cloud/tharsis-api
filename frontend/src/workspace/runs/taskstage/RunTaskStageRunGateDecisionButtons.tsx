import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import CheckIcon from '@mui/icons-material/Check';
import { Button, ButtonGroup, Menu, MenuItem, TextField, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useState } from 'react';
import { useMutation } from 'react-relay/hooks';
import ConfirmationDialog from '../../../common/ConfirmationDialog';
import { MutationError } from '../../../common/error';
import RunTaskStageOverrideRunGateDialog from './RunTaskStageOverrideRunGateDialog';
import { RunTaskStageRunGateDecisionButtonsApproveMutation } from './__generated__/RunTaskStageRunGateDecisionButtonsApproveMutation.graphql';

interface Props {
    gateId: string;
    // Whether overriding is currently possible — only a soft-failed check can be overridden. When
    // false the Approve button has no dropdown, since Override is the only entry in it.
    canOverride: boolean;
    // Connections the gate should be dropped from once this caller has decided. Set by the approvals
    // inbox, where a decided gate no longer belongs; empty on the run details page, which shows the
    // gate whatever its state — an empty list makes the @deleteEdge below a no-op.
    connectionIds?: readonly string[];
    // Called after a decision the API accepted, so a list can adjust its own counts.
    onDecided?: () => void;
    onError: (error: MutationError) => void;
}

// RunTaskStageRunGateDecisionButtons is the Approve / Reject pair for a pending run gate. Rejection is advisory
// on the API: it records the decision but leaves the gate pending so another eligible approver can
// still approve, which is why Reject is styled as a secondary action rather than a destructive one.
function RunTaskStageRunGateDecisionButtons({ gateId, canOverride, connectionIds, onDecided, onError }: Props) {
    const [commitApprove, commitApproveInFlight] = useMutation<RunTaskStageRunGateDecisionButtonsApproveMutation>(graphql`
        mutation RunTaskStageRunGateDecisionButtonsApproveMutation($input: ApproveRunGateInput!, $connections: [ID!]!) {
            approveRunGate(input: $input) {
                runGate {
                    id @deleteEdge(connections: $connections)
                    status
                    approvals {
                        id
                        decision
                        comment
                        createdBy
                        coveredRules
                        metadata {
                            createdAt
                        }
                        user {
                            id
                            username
                            email
                        }
                        serviceAccount {
                            id
                            name
                            resourcePath
                        }
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const [decision, setDecision] = useState<'APPROVE' | 'REJECT' | null>(null);
    const [comment, setComment] = useState('');
    const [menuAnchorEl, setMenuAnchorEl] = useState<HTMLElement | null>(null);
    const [showOverrideDialog, setShowOverrideDialog] = useState(false);

    const submitDecision = (confirm?: boolean) => {
        if (!confirm || !decision) {
            setDecision(null);
            setComment('');
            return;
        }
        commitApprove({
            variables: { input: { gateId, decision, comment }, connections: connectionIds ? [...connectionIds] : [] },
            onCompleted: response => {
                setDecision(null);
                setComment('');
                if (response.approveRunGate.problems.length) {
                    onError({
                        severity: 'warning',
                        message: response.approveRunGate.problems.map(problem => problem.message).join('; ')
                    });
                    return;
                }
                onDecided?.();
            },
            onError: error => {
                setDecision(null);
                setComment('');
                onError({
                    severity: 'error',
                    message: `Unexpected Error Occurred: ${error.message}`
                });
            }
        });
    };

    return (
        <>
            <Button size="small" variant="outlined" color="inherit" onClick={() => setDecision('REJECT')}>
                Reject
            </Button>
            {canOverride ? (
                <ButtonGroup variant="contained" color="primary" size="small">
                    <Button startIcon={<CheckIcon />} onClick={() => setDecision('APPROVE')}>
                        Approve
                    </Button>
                    <Button
                        aria-label="more gate options menu"
                        aria-haspopup="menu"
                        onClick={event => setMenuAnchorEl(event.currentTarget)}
                    >
                        <ArrowDropDownIcon fontSize="small" />
                    </Button>
                </ButtonGroup>
            ) : (
                <Button
                    size="small"
                    variant="contained"
                    startIcon={<CheckIcon />}
                    onClick={() => setDecision('APPROVE')}
                >
                    Approve
                </Button>
            )}
            <Menu
                anchorEl={menuAnchorEl}
                open={Boolean(menuAnchorEl)}
                onClose={() => setMenuAnchorEl(null)}
                anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
                <MenuItem
                    onClick={() => {
                        setMenuAnchorEl(null);
                        setShowOverrideDialog(true);
                    }}
                >
                    Override Gate
                </MenuItem>
            </Menu>
            {/* Rendered outside the Menu so closing the menu doesn't unmount the dialog. */}
            {showOverrideDialog && <RunTaskStageOverrideRunGateDialog
                gateId={gateId}
                onClose={() => setShowOverrideDialog(false)}
                onError={onError}
            />}
            {decision && <ConfirmationDialog
                title={decision === 'APPROVE' ? 'Approve Gate' : 'Reject Gate'}
                maxWidth="sm"
                confirmColor={decision === 'APPROVE' ? 'primary' : 'error'}
                confirmLabel={decision === 'APPROVE' ? 'Approve' : 'Reject'}
                confirmInProgress={commitApproveInFlight}
                onConfirm={() => submitDecision(true)}
                onClose={() => submitDecision()}
            >
                <Typography variant="subtitle2" gutterBottom>Comment (optional)</Typography>
                <TextField
                    autoComplete="off"
                    fullWidth
                    multiline
                    minRows={2}
                    size="small"
                    placeholder="Add a comment"
                    value={comment}
                    onChange={(e) => setComment(e.target.value)}
                />
            </ConfirmationDialog>}
        </>
    );
}

export default RunTaskStageRunGateDecisionButtons;
