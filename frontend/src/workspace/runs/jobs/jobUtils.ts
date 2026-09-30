import moment from 'moment';
import { useTheme } from '@mui/material/styles';
import { shortDuration } from '../../../common/duration';

export function jobDuration(timestamps: { runningAt: unknown; finishedAt: unknown } | null | undefined): moment.Duration | null {
    if (!timestamps?.finishedAt || !timestamps?.runningAt) {
        return null;
    }

    return moment.duration(
        moment(timestamps.finishedAt as moment.MomentInput).diff(
            moment(timestamps.runningAt as moment.MomentInput)
        )
    );
}

export function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
    const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);

    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

export const JOB_STATUS_LABELS: Record<string, string> = {
    canceling: 'Canceling',
    canceled: 'Canceled',
    failed: 'Failed',
    finished: 'Completed',
    running: 'Running',
    pending: 'Pending',
    queued: 'Queued',
};

export const JOB_TYPE_LABELS: Record<string, string> = {
    plan: 'Plan',
    apply: 'Apply',
    opa: 'Policy',
};

export function useJobStatusColor(status: string): string {
    const theme = useTheme();

    return theme.palette.jobStatus[status.toLowerCase() as keyof typeof theme.palette.jobStatus]
        ?? theme.palette.runStatus.unknown;
}

export function humanizeDurationBetween(from: unknown, to: unknown): string | null {
    if (!from || !to) {
        return null;
    }

    const ms = moment(to as moment.MomentInput).diff(moment(from as moment.MomentInput));

    return shortDuration(ms);
}
