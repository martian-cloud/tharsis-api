// Self-hosted Monaco setup.
//
// The default @monaco-editor/react loader fetches Monaco (and its web workers)
// from a CDN (jsDelivr) at runtime. Our Content-Security-Policy uses
// `script-src 'self'` (see index.html), which blocks that CDN load. Instead we
// bundle `monaco-editor` locally and serve its worker from our own origin.
//
// CSP fit: the bundled editor is served as app scripts (`script-src 'self'`),
// and the worker is emitted/loaded from our origin or as a blob, both allowed
// by `worker-src 'self' blob:`.
//
// Import this module for its side effects BEFORE any <Editor> renders (it is
// imported at the top of the pages that use Monaco).
import { loader } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
// Vite compiles these worker entries into same-origin assets (see vite/client
// types for the `?worker` suffix).
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import { registerRegoLanguage } from './regoLanguage';

declare global {
    interface Window {
        MonacoEnvironment?: monaco.Environment;
    }
}

// Serve the matching worker per language service. Our custom `rego` language uses
// a Monarch tokenizer (highlighting runs on the main thread) so it only needs the
// base editor worker, but the JSON language service registers providers (folding,
// document symbols, validation) that delegate to a dedicated `json` worker — if it
// isn't served, those requests fail with "Missing requestHandler or method".
window.MonacoEnvironment = {
    getWorker: (_workerId, label) => (label === 'json' ? new JsonWorker() : new EditorWorker()),
};

// Register the custom Rego language (syntax highlighting) on the bundled instance.
registerRegoLanguage(monaco);

// Point @monaco-editor/react at the locally bundled monaco instead of the CDN.
loader.config({ monaco });
