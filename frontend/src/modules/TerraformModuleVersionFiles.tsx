import { useCallback, useContext } from 'react';
import AuthServiceContext from '../auth/AuthServiceContext';
import AuthenticationService from '../auth/AuthenticationService';
import ArchiveFileBrowser from '../archive/ArchiveFileBrowser';
import { ArchiveTooLargeError, MAX_DOWNLOAD_BYTES } from '../archive/tarball';
import cfg from '../common/config';

interface ModuleVersionParams {
    registryNamespace: string;
    name: string;
    system: string;
    version: string;
}

// fetchModulePackage downloads the module version tarball as an ArrayBuffer using the same two-step
// flow as the download button: an authenticated request returns a presigned URL via X-Terraform-Get.
export async function fetchModulePackage(
    authService: AuthenticationService,
    { registryNamespace, name, system, version }: ModuleVersionParams
): Promise<ArrayBuffer> {
    let response = await authService.fetchWithAuth(
        `${cfg.apiUrl}/v1/module-registry/modules/${registryNamespace}/${name}/${system}/${version}/download`,
        { method: 'GET' }
    );

    if (!response.ok) {
        throw new Error(`request for module download url returned status ${response.status}`);
    }

    const downloadUrl = response.headers.get('X-Terraform-Get');
    if (!downloadUrl) {
        throw new Error('response for module download url is missing header X-Terraform-Get');
    }

    response = await fetch(downloadUrl, { method: 'GET' });

    if (!response.ok) {
        throw new Error(`requested to download module returned status ${response.status}`);
    }

    const contentLength = Number(response.headers.get('content-length'));
    if (!contentLength) {
        throw new Error('module archive response is missing a content-length header');
    }

    if (contentLength > MAX_DOWNLOAD_BYTES) {
        throw new ArchiveTooLargeError('module archive is too large to preview');
    }

    return response.arrayBuffer();
}

interface Props {
    module: ModuleVersionParams;
}

function TerraformModuleVersionFiles({ module }: Props) {
    const authService = useContext<AuthenticationService>(AuthServiceContext);

    const load = useCallback(() => fetchModulePackage(authService, module), [authService, module]);

    return <ArchiveFileBrowser load={load} preferredFile="main.tf" />;
}

export default TerraformModuleVersionFiles;
