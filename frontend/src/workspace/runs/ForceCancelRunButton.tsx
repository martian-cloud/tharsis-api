import React, { useState } from 'react'
import { LoadingButton } from '@mui/lab';
import { Alert, AlertTitle, Stack, TextField, Typography } from '@mui/material'
import { useFragment, useMutation } from 'react-relay/hooks'
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import ConfirmationDialog from '../../common/ConfirmationDialog';
import { ForceCancelRunButtonFragment_run$key } from './__generated__/ForceCancelRunButtonFragment_run.graphql'
import { ForceCancelRunButtonCancelRunMutation } from './__generated__/ForceCancelRunButtonCancelRunMutation.graphql'

interface Props {
    fragmentRef: ForceCancelRunButtonFragment_run$key
}

function ForceCancelRunButton(props: Props) {
    const [openDialog, setOpenDialog] = useState(false)
    const [confirmInput, setConfirmInput] = useState('')
    const { enqueueSnackbar } = useSnackbar();

    const data = useFragment<ForceCancelRunButtonFragment_run$key>(
        graphql`
        fragment ForceCancelRunButtonFragment_run on Run {
            id
            workspace {
                fullPath
            }
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
                    setConfirmInput('')
                    if (data.cancelRun.problems.length) {
                        enqueueSnackbar(data.cancelRun.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
                    }
                },
                onError: error => {
                    setOpenDialog(false)
                    setConfirmInput('')
                    enqueueSnackbar(`An unexpected error occurred: ${error.message}`, { variant: 'error' });
                }
            })
        } else {
            setOpenDialog(false)
            setConfirmInput('')
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
            {openDialog && (
                <ConfirmationDialog
                    title="Force Cancel Run"
                    maxWidth="sm"
                    confirmLabel="Force Cancel"
                    confirmDisabled={data.workspace.fullPath !== confirmInput}
                    confirmInProgress={commitForceCancelRunInFlight}
                    onConfirm={() => forceCancelRun(true)}
                    onClose={() => forceCancelRun()}
                >
                    <Alert sx={{ mb: 2 }} severity="warning">
                        <AlertTitle>Warning</AlertTitle>
                        Force cancelling this run may result in a stale workspace state.
                    </Alert>
                    <Typography variant="subtitle2">Enter the following to confirm force cancelling this run:</Typography>
                    <SyntaxHighlighter style={prismTheme} customStyle={{ fontSize: 14, marginBottom: 14 }}>{data.workspace.fullPath}</SyntaxHighlighter>
                    <TextField
                        autoComplete="off"
                        fullWidth
                        size="small"
                        placeholder={data.workspace.fullPath}
                        value={confirmInput}
                        onChange={(e) => setConfirmInput(e.target.value)}
                    />
                </ConfirmationDialog>
            )}
        </Stack>
    )
}

export default ForceCancelRunButton
