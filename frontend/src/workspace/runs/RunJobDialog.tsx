import React, { Suspense, useMemo } from 'react';
import CloseIcon from '@mui/icons-material/Close';
import { Box, Button, Chip, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, Link as MuiLink, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Tooltip, Typography } from "@mui/material";
import { useFragment } from 'react-relay/hooks';
import IconButton from '@mui/material/IconButton';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import Link from '../../routes/Link';
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
                    <TableContainer sx={{
                        borderTop: `1px solid ${theme.palette.divider}`,
                        borderLeft: `1px solid ${theme.palette.divider}`,
                        borderRight: `1px solid ${theme.palette.divider}`,
                        borderBottom: `1px solid ${theme.palette.divider}`,
                        borderBottomLeftRadius: 4,
                        borderBottomRightRadius: 4,
                    }}>
                        <Table
                            sx={{ minWidth: 650, tableLayout: 'fixed' }}
                            aria-label="runner jobs"
                        >
                            <TableHead>
                                <TableRow>
                                    <TableCell>Status</TableCell>
                                    <TableCell>ID</TableCell>
                                    <TableCell>Tags</TableCell>
                                    <TableCell>Runner</TableCell>
                                    <TableCell>Duration</TableCell>
                                    <TableCell>Created</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                <TableRow sx={{ '&:last-child td, &:last-child th': { border: 0 } }}>
                                    <TableCell>
                                        <JobStatusChip status={data.status} onClick={onClose} />
                                    </TableCell>
                                    <TableCell>
                                        <MuiLink
                                            sx={{ cursor: 'pointer' }}
                                            color="textPrimary" underline="hover"
                                            onClick={() => onClose()}
                                        >{data.id.substring(0, 8)}...
                                        </MuiLink>
                                    </TableCell>
                                    <TableCell>
                                        {(data.tags && data.tags.length > 0) ? <Stack direction="row" spacing={1}>
                                            {data.tags.map((tag: any) => <Chip key={tag} size="small" color="secondary" label={tag} />)}
                                        </Stack> : <Typography variant="body2" color="textSecondary">None</Typography>}
                                    </TableCell>
                                    <TableCell>
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
                                        {data.timestamps.pendingAt &&
                                            <Typography
                                                component="div"
                                                variant="caption"
                                            >claimed job {moment(data.timestamps.pendingAt as moment.MomentInput).fromNow()}
                                            </Typography>}
                                    </TableCell>
                                    <TableCell>
                                        {duration ? humanizeDuration(duration.asMilliseconds()) : '--'}
                                    </TableCell>
                                    <TableCell>
                                        <Tooltip title={data.metadata.createdAt}>
                                            <Box>{moment(data.metadata.createdAt as moment.MomentInput).fromNow()}</Box>
                                        </Tooltip>
                                    </TableCell>
                                </TableRow >
                            </TableBody>
                        </Table>
                    </TableContainer>
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
