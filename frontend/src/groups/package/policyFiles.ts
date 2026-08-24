import { nanoid } from 'nanoid';

/**
 * PolicyFile is a single authored policy file (a tar entry: filename + text content).
 *
 * This lives apart from PackageFilesEditor because PackageVersionFiles needs the shape and the
 * factory but must not pull in Monaco, which the editor imports for its side effects.
 */
export interface PolicyFile {
    name: string;
    content: string;
    /** Row key, stable for the lifetime of the row. Not part of the uploaded tarball. */
    _id: string;
}

/** A file row. Every row needs its own _id, so this has to be a function rather than a literal. */
export function newPolicyFile(name: string, content: string): PolicyFile {
    return { name, content, _id: nanoid() };
}
