import { useCallback, useContext } from 'react';
import ArchiveFileBrowser from '../archive/ArchiveFileBrowser';
import { ArchiveTooLargeError, MAX_DOWNLOAD_BYTES, decodeText, extractTarGz } from '../archive/tarball';
import AuthServiceContext from '../auth/AuthServiceContext';
import AuthenticationService from '../auth/AuthenticationService';
import cfg from '../common/config';
import { PolicyFile, newPolicyFile } from '../groups/package/policyFiles';

// fetchPackagePackage downloads the package version tarball as an ArrayBuffer using the same
// two-step flow as the module registry: an authenticated request returns a presigned URL via the
// X-Download-Url header, which is then fetched directly (unauthenticated). This avoids following a
// redirect with the Authorization header attached, which would trigger a CORS preflight the object
// store rejects.
export async function fetchPackagePackage(
    authService: AuthenticationService,
    versionId: string
): Promise<ArrayBuffer> {
    let response = await authService.fetchWithAuth(
        `${cfg.apiUrl}/v1/package-registry/versions/${versionId}/download`,
        { method: 'GET' }
    );

    if (!response.ok) {
        throw new Error(`request for package download url returned status ${response.status}`);
    }

    const downloadUrl = response.headers.get('X-Download-Url');
    if (!downloadUrl) {
        throw new Error('response for package download url is missing header X-Download-Url');
    }

    response = await fetch(downloadUrl, { method: 'GET' });

    if (!response.ok) {
        throw new Error(`request to download package version returned status ${response.status}`);
    }

    const contentLength = Number(response.headers.get('content-length'));
    if (contentLength && contentLength > MAX_DOWNLOAD_BYTES) {
        throw new ArchiveTooLargeError('package archive is too large to preview');
    }

    return response.arrayBuffer();
}

// fetchPolicyFiles downloads a package version and extracts it into editable files. It backs both
// version-authoring flows: editing a version's files in place, and seeding a new version with what the
// latest version already contains.
export async function fetchPolicyFiles(
    authService: AuthenticationService,
    versionId: string
): Promise<PolicyFile[]> {
    const buffer = await fetchPackagePackage(authService, versionId);
    const archiveFiles = await extractTarGz(buffer);

    return archiveFiles.map(file => newPolicyFile(file.path, decodeText(file.data).fullText));
}

interface Props {
    versionId: string;
}

function PackageVersionFiles({ versionId }: Props) {
    const authService = useContext<AuthenticationService>(AuthServiceContext);

    const load = useCallback(() => fetchPackagePackage(authService, versionId), [authService, versionId]);

    // A package is a handful of Rego files, and the details page gives the browser a narrow column
    // beside the details sidebar, so the file picker lives in the header rather than a tree.
    return <ArchiveFileBrowser load={load} preferredFile="policy.rego" filePicker="dropdown" />;
}

export default PackageVersionFiles;
