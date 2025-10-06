import AuthenticationService from "../../../auth/AuthenticationService";
import cfg from "../../../common/config";

export const uploadConfigVersionPackage = async (
    file: Blob,
    workspaceId: string,
    configVersionId: string,
    authService: AuthenticationService
) => {
    const fileData = await file.arrayBuffer();

    const requestOptions = {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/octet-stream'
        },
        body: new Uint8Array(fileData),
    };

    const response = await authService.fetchWithAuth( // nosemgrep: nodejs_scan.javascript-ssrf-rule-node_ssrf
        `${cfg.apiUrl}/tfe/v2/workspaces/${workspaceId}/configuration-versions/${configVersionId}/upload`,
        requestOptions
    );

    if (!response.ok) {
        throw new Error(`Error uploading file ${response}`);
    }
};
