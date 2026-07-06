import React, { Suspense, useMemo } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import { Box, Button, Chip, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, Link as MuiLink, Stack, Typography } from "@mui/material";
import { ResponsiveRow, ResponsiveTable } from '../../common/ResponsiveTable';
import { useFragment } from 'react-relay/hooks';
import IconButton from '@mui/material/IconButton';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import Link from '../../routes/Link';
import Timestamp from '../../common/Timestamp';
import moment from 'moment';
import humanizeDuration from 'humanize-duration';
import JobStatusChip from './JobStatusChip';
import { RunJobDialog_currentJob$key } from './__generated__/RunJobDialog_currentJob.graphql';

interface Props {
    fragmentRef: RunJobDialog_currentJob$key
    onClose: (confirm?: boolean) => void
}

function RunJobDialog(props: Props) {
    const { fragmentRef, onClose } = props;
    const theme = useTheme();
    const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

    const data = useFragment<RunJobDialog_currentJob$key>(
        graphql`
        fragment RunJobDialog_currentJob on Job
        {
            id
            status
            tags
            runner {
                id
                name
                type
                groupPath
            }
            runnerPath
            metadata {
                createdAt
            }
            timestamps {
                pendingAt
                runningAt
                finishedAt
            }
        }
        `, fragmentRef
    );

    const timestamps = data.timestamps;
    const duration = useMemo(() => timestamps?.finishedAt ?
        moment.duration(moment(timestamps.finishedAt as moment.MomentInput).diff(moment(timestamps.runningAt as moment.MomentInput))) : null, [timestamps]);

    return (
        <Dialog
            open
            maxWidth="lg"
            fullWidth
            fullScreen={fullScreen}
        >
            <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                Job Details
                <IconButton
                    color="inherit"
                    size="small"
                    onClick={() => onClose()}
                >
                    <CloseIcon />
                </IconButton>
            </DialogTitle>
            <DialogContent dividers sx={{ flex: 1, padding: 2, minHeight: 600, display: 'flex', flexDirection: 'column' }}>
                <Suspense fallback={<Box
                    sx={{
                        position: 'absolute',
                        top: 0,
                        left: 0,
                        width: '100%',
                        minHeight: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center'
                    }}>
                    <CircularProgress />
                </Box>}>
                    <ResponsiveTable
                        ariaLabel="runner jobs"
                        columns={[
                            { label: 'Status' },
                            { label: 'ID' },
                            { label: 'Tags' },
                            { label: 'Runner' },
                            { label: 'Duration' },
                            { label: 'Created' },
                        ]}
                    >
                        <ResponsiveRow cells={[
                            { primary: true, content: <JobStatusChip status={data.status} onClick={onClose} /> },
                            {
                                label: 'ID', content: <MuiLink
                                    sx={{ cursor: 'pointer' }}
                                    color="textPrimary" underline="hover"
                                    onClick={() => onClose()}
                                >{data.id.substring(0, 8)}...
                                </MuiLink>
                            },
                            {
                                label: 'Tags', content: (data.tags && data.tags.length > 0) ? <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                                    {data.tags.map((tag: any) => <Chip key={tag} size="small" color="secondary" label={tag} />)}
                                </Stack> : <Typography variant="body2" color="textSecondary">None</Typography>
                            },
                            {
                                label: 'Runner', content: <Box>
                                    {data.runner ? <Link
                                        color="primary"
                                        sx={{ fontWeight: 500 }}
                                        to={data.runner.type === 'shared' ?
                                            `/admin/runners/${data.runner.id}`
                                            : `/groups/${data.runner.groupPath}/-/runners/${data.runner.id}`
                                        }
                                    >
                                        {data.runner.name}
                                    </Link> : <React.Fragment>--</React.Fragment>}
                                    {!data.runner && data.runnerPath && <React.Fragment>{data.runnerPath} (deleted)</React.Fragment>}
                                    {data.timestamps?.pendingAt &&
                                        <Typography
                                            component="div"
                                            variant="caption"
                                        >claimed job <Timestamp timestamp={data.timestamps.pendingAt as string} />
                                        </Typography>}
                                </Box>
                            },
                            { label: 'Duration', content: duration ? humanizeDuration(duration.asMilliseconds()) : '--' },
                            { label: 'Created', content: <Timestamp timestamp={data.metadata.createdAt as string} /> },
                        ]} />
                    </ResponsiveTable>
                </Suspense>
            </DialogContent>
            {!fullScreen && <DialogActions>
                <Button color="inherit" onClick={() => onClose()}>
                    Close
                </Button>
            </DialogActions>}
        </Dialog>
    );
}

export default RunJobDialog;
