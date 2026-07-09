// Shared helpers for rendering Terraform check result statuses.

export function getCheckStatusLabel(status: string): string {
    switch (status) {
        case 'pass': return 'Pass';
        case 'fail': return 'Fail';
        case 'error': return 'Error';
        case 'unknown': return 'Unknown';
        default: return status;
    }
}

export function getCheckStatusTooltip(status: string): string {
    switch (status) {
        case 'pass': return 'The check assertion condition evaluated to true';
        case 'fail': return 'The check assertion condition evaluated to false';
        case 'error': return 'Terraform could not evaluate the check condition';
        case 'unknown': return 'The check result could not be determined';
        default: return '';
    }
}

export function collectFailureMessages(objects: readonly { readonly failureMessages: readonly string[] }[]): readonly string[] {
    return objects.flatMap((obj) => obj.failureMessages);
}
