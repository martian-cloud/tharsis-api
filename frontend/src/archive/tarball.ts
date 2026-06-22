import { parseTar } from 'nanotar';

// Per-file preview caps: the highlighter builds a node per token, so large files are truncated.
export const MAX_RENDER_BYTES = 128 * 1024;
export const MAX_RENDER_LINES = 500;

// Bounds on untrusted archives: download size cap, decompression-bomb size cap, and
// (un-virtualized) tree file-count cap.
export const MAX_DOWNLOAD_BYTES = 50 * 1024 * 1024;
export const MAX_DECOMPRESSED_BYTES = 50 * 1024 * 1024;
export const MAX_FILES = 2000;

// Thrown when an archive exceeds MAX_DOWNLOAD_BYTES and is too large to preview.
export class ArchiveTooLargeError extends Error {}

export interface ArchiveFile {
    path: string;
    size: number;
    data: Uint8Array;
}

// extractTarGz gunzips and untars, returning regular files only (dedup paths, bounded size/count).
export async function extractTarGz(data: ArrayBuffer): Promise<ArchiveFile[]> {
    const decompressed = await gunzip(data, MAX_DECOMPRESSED_BYTES);
    const items = parseTar(decompressed);
    const seen = new Set<string>();
    const files: ArchiveFile[] = [];

    for (const item of items) {
        if (item.type !== 'file' || !item.name || !item.data) {
            continue;
        }

        const path = item.name.replace(/^\.\//, '');
        if (seen.has(path)) {
            continue;
        }

        if (files.length >= MAX_FILES) {
            throw new Error('archive contains too many files to preview');
        }

        seen.add(path);
        files.push({ path, size: item.size, data: item.data as Uint8Array });
    }

    return files;
}

// gunzip decompresses gzip, aborting past maxBytes to bound decompression bombs.
async function gunzip(data: ArrayBuffer, maxBytes: number): Promise<Uint8Array> {
    const stream = new Blob([data]).stream().pipeThrough(new window.DecompressionStream('gzip'));
    const reader = stream.getReader();
    const chunks: Uint8Array[] = [];
    let total = 0;

    for (; ;) {
        const { done, value } = await reader.read();
        if (done) {
            break;
        }

        total += value.length;
        if (total > maxBytes) {
            await reader.cancel();
            throw new Error('archive exceeds the maximum supported size');
        }

        chunks.push(value);
    }

    const result = new Uint8Array(total);
    let offset = 0;
    for (const chunk of chunks) {
        result.set(chunk, offset);
        offset += chunk.length;
    }

    return result;
}

const languageByExtension: Record<string, string> = {
    tf: 'hcl',
    hcl: 'hcl',
    tfvars: 'hcl',
    sh: 'bash',
    bash: 'bash',
    json: 'json',
    md: 'markdown',
    yml: 'yaml',
    yaml: 'yaml',
    py: 'python',
    js: 'javascript',
    ts: 'typescript',
    go: 'go',
    toml: 'toml',
};

// languageForFile maps a file path to a Prism language for syntax highlighting.
export function languageForFile(path: string): string {
    const extension = path.split('.').pop()?.toLowerCase() ?? '';
    return languageByExtension[extension] ?? 'text';
}

export interface DecodedFile {
    // text is the truncated preview; fullText is the complete file (for copy).
    text: string;
    fullText: string;
    truncated: boolean;
    binary: boolean;
}

// decodeText decodes UTF-8 once, flagging binary content and truncating the preview.
export function decodeText(data: Uint8Array): DecodedFile {
    const empty = { text: '', fullText: '', truncated: false, binary: true };

    // NUL byte: cheap binary signal on a bounded prefix
    if (data.subarray(0, MAX_RENDER_BYTES).includes(0)) {
        return empty;
    }

    const fullText = new window.TextDecoder('utf-8').decode(data);
    // replacement char => invalid UTF-8 => binary
    if (fullText.includes('�')) {
        return empty;
    }

    let text = fullText;
    let truncated = false;
    if (text.length > MAX_RENDER_BYTES) {
        text = text.slice(0, MAX_RENDER_BYTES);
        truncated = true;
    }

    const lines = text.split('\n');
    if (lines.length > MAX_RENDER_LINES) {
        text = lines.slice(0, MAX_RENDER_LINES).join('\n');
        truncated = true;
    }

    return { text, fullText, truncated, binary: false };
}
