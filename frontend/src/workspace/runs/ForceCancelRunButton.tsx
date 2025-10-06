import React, { useState } from 'react'
import { LoadingButton } from '@mui/lab';
import { Alert, AlertTitle, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material'
import { useFragment, useMutation } from 'react-relay/hooks'
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { ForceCancelRunButtonFragment_run$key } from './__generated__/ForceCancelRunButtonFragment_run.graphql'
import { ForceCancelRunButtonCancelRunMutation } from './__generated__/ForceCancelRunButtonCancelRunMutation.graphql'
import { ForceCancelRunButtonDialogFragment_run$key } from './__generated__/ForceCancelRunButtonDialogFragment_run.graphql'

interface Props {
    fragmentRef: ForceCancelRunButtonFragment_run$key
}

interface ForceCancelDialogProps {
    fragmentRef: ForceCancelRunButtonDialogFragment_run$key
    open: boolean
    closeDialog: () => void
    onClose: (confirm?: boolean) => void
    cancelInProgress: boolean
}

function ForceCancelConfirmationDialog(props: ForceCancelDialogProps) {
    const { fragmentRef, open, closeDialog, onClose, cancelInProgress } = props
    const [deleteInput, setDeleteInput] = useState<string>('')

    const data = useFragment<ForceCancelRunButtonDialogFragment_run$key>(
        graphql`
        fragment ForceCancelRunButtonDialogFragment_run on Run
        {
            workspace {
                fullPath
            }
        }
        `, fragmentRef
    )

    return (
        <Dialog
            keepMounted
            maxWidth="sm"
            open={open}
        >
            <DialogTitle>Force Cancel Run</DialogTitle>
            <DialogContent>
                <Alert sx={{ mb: 2 }} severity="warning">
                    <AlertTitle>Warning</AlertTitle>
                    Force cancelling this run may result in a stale workspace state.
                </Alert>
                <Typography variant="subtitle2">Enter the following to confirm force cancelling this run:</Typography>
                <SyntaxHighlighter style={prismTheme} customStyle={{ fontSize: 14, marginBottom: 14 }} children={data.workspace.fullPath} />
                <TextField
                    autoComplete="off"
                    fullWidth
                    size="small"
                    placeholder={data.workspace.fullPath}
                    value={deleteInput}
                    onChange={(event: any) => setDeleteInput(event.target.value)}
                ></TextField>
            </DialogContent>
            <DialogActions>
                <Button color="inherit"
                    onClick={() => {
                        closeDialog()
                        setDeleteInput('')
                    }}>Cancel
                </Button>
                <LoadingButton
                    color="error"
                    variant="outlined"
                    loading={cancelInProgress}
                    disabled={data.workspace.fullPath !== deleteInput}
                    onClick={() => {
                        onClose(true)
                        setDeleteInput('')
                    }}
                >Force Cancel</LoadingButton>
            </DialogActions>
        </Dialog>
    )
}

function ForceCancelRunButton(props: Props) {
    const [openDialog, setOpenDialog] = useState(false)
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<ForceCancelRunButtonFragment_run$key>(
        graphql`
        fragment ForceCancelRunButtonFragment_run on Run {
            id
            ...ForceCancelRunButtonDialogFragment_run
        }
    `, props.fragmentRef)

    const [commitForceCancelRun, commitForceCancelRunInFlight] = useMutation<ForceCancelRunButtonCancelRunMutation>(graphql`
        mutation ForceCancelRunButtonCancelRunMutation($input: CancelRunInput!) {
            cancelRun(input: $input) {
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

    const forceCancelRun = (confirm?: boolean) => {
        if (confirm) {
            commitForceCancelRun({
                variables: {
                    input: {
                        runId: data.id,
                        force: true
                    },
                },
                onCompleted: data => {
                    setOpenDialog(false)

                    if (data.cancelRun.problems.length) {
                        enqueueSnackbar(data.cancelRun.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    }
                },
                onError: error => {
                    setOpenDialog(false)
                    enqueueSnackbar(`An unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            })
        }
        else {
            setOpenDialog(false)
        }
    }

    return (
        <Stack direction="row" spacing={2}>
            <LoadingButton
                loading={commitForceCancelRunInFlight}
                size="small"
                variant="outlined"
                color="warning"
                onClick={() => setOpenDialog(true)}
            >
                Force Cancel
            </LoadingButton>
            <ForceCancelConfirmationDialog
                fragmentRef={data}
                cancelInProgress={commitForceCancelRunInFlight}
                open={openDialog}
                closeDialog={() => setOpenDialog(false)}
                onClose={forceCancelRun}
            />
        </Stack>
    )
}

export default ForceCancelRunButton
