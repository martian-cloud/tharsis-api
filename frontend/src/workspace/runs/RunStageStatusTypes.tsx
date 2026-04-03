import NotStartedIcon from '@mui/icons-material/AdjustOutlined';
import CheckCircleIcon from '@mui/icons-material/CheckCircleOutline';
import ErrorIcon from '@mui/icons-material/Error';
import PendingIcon from '@mui/icons-material/PauseCircleOutline';
import InProgressIcon from '@mui/icons-material/TimelapseOutlined';
import WarningIcon from '@mui/icons-material/Warning';
import { SxProps, Theme } from '@mui/material';

interface IconProps {
    sx?: SxProps<Theme>;
}

export default {
    created: {
        label: 'Not Started',
        color: 'runStatus.created',
        icon: ({sx}: IconProps) => <NotStartedIcon sx={{ ...sx, color: 'runStatus.created' }} />,
        tooltip: 'has not started'
    },
    canceled: {
        label: 'Canceled',
        color: 'runStatus.canceled',
        icon: ({sx}: IconProps) => <WarningIcon sx={{ ...sx, color: 'runStatus.canceled' }} />,
        tooltip: 'was canceled while in progress'
    },
    errored: {
        label: 'Failed',
        color: 'runStatus.errored',
        icon: ({sx}: IconProps) => <ErrorIcon sx={{ ...sx, color: 'runStatus.errored' }} />,
        tooltip: 'returned an error'
    },
    finished: {
        label: 'Completed',
        color: 'runStatus.finished',
        icon: ({sx}: IconProps) => <CheckCircleIcon sx={{ ...sx, color: 'runStatus.finished' }} />,
        tooltip: 'has completed'
    },
    pending: {
        label: 'Pending',
        color: 'runStatus.pending',
        icon: ({sx}: IconProps) => <PendingIcon sx={{ ...sx, color: 'runStatus.pending' }} />,
        tooltip: 'is pending'
    },
    queued: {
        label: 'Queued',
        color: 'runStatus.queued',
        icon: ({sx}: IconProps) => <PendingIcon sx={{ ...sx, color: 'runStatus.queued' }} />,
        tooltip: 'is queued'
    },
    running: {
        label: 'In Progress',
        color: 'runStatus.running',
        icon: ({sx}: IconProps) => <InProgressIcon sx={{ ...sx, color: 'runStatus.running' }} />,
        tooltip: 'is in progress'
    },
    applied: {
        label: 'Applied',
        color: 'runStatus.applied',
        icon: ({sx}: IconProps) => <CheckCircleIcon sx={{ ...sx, color: 'runStatus.applied' }} />,
        tooltip: 'has applied'
    },
    apply_queued: {
        label: 'Apply Queued',
        color: 'runStatus.apply_queued',
        icon: ({sx}: IconProps) => <PendingIcon sx={{ ...sx, color: 'runStatus.apply_queued' }} />,
        tooltip: 'is queued'
    },
    applying: {
        label: 'Applying',
        color: 'runStatus.applying',
        icon: ({sx}: IconProps) => <InProgressIcon sx={{ ...sx, color: 'runStatus.applying' }} />,
        tooltip: 'is applying'
    },
    plan_queued: {
        label: 'Plan Queued',
        color: 'runStatus.plan_queued',
        icon: ({sx}: IconProps) => <PendingIcon sx={{ ...sx, color: 'runStatus.plan_queued' }} />,
        tooltip: 'is queued'
    },
    planned: {
        label: 'Applying',
        color: 'runStatus.planned',
        icon: ({sx}: IconProps) => <InProgressIcon sx={{ ...sx, color: 'runStatus.planned' }} />,
        tooltip: 'is planned'
    },
    planned_and_finished: {
        label: 'Planned and Finished',
        color: 'runStatus.planned_and_finished',
        icon: ({sx}: IconProps) => <CheckCircleIcon sx={{ ...sx, color: 'runStatus.planned_and_finished' }} />,
        tooltip: 'has planned and finished'
    },
    planning: {
        label: 'Planning',
        color: 'runStatus.planning',
        icon: ({sx}: IconProps) => <InProgressIcon sx={{ ...sx, color: 'runStatus.planning' }} />,
        tooltip: 'is planning'
    }
} as any;
