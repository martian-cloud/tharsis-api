// Shared helpers for rendering Terraform check result statuses.

export function getCheckStatusLabel(status: string): string {
    switch (status) {
        case 'PASS': return 'Pass';
        case 'FAIL': return 'Fail';
        case 'ERROR': return 'Error';
        case 'UNKNOWN': return 'Unknown';
        default: return status;
    }
}

export function getCheckStatusTooltip(status: string): string {
    switch (status) {
        case 'PASS': return 'The check assertion condition evaluated to true';
        case 'FAIL': return 'The check assertion condition evaluated to false';
        case 'ERROR': return 'Terraform could not evaluate the check condition';
        case 'UNKNOWN': return 'The check result could not be determined';
        default: return '';
    }
}

export function collectFailureMessages(objects: readonly { readonly failureMessages: readonly string[] }[]): readonly string[] {
    return objects.flatMap((obj) => obj.failureMessages);
}
