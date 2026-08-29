// Base64 decoding helpers.
//
// atob alone is byte-oriented: it returns one character per decoded byte, so any multi-byte UTF-8
// sequence — an accented name, a CJK character, an emoji — is split into separate characters and the
// text comes back mojibake ("André" reads as "AndrÃ©"). Everything the API hands us base64-encoded is
// UTF-8 text, so the bytes have to be run back through a UTF-8 decoder.

// decodeBase64Utf8 decodes base64-encoded UTF-8 text.
export function decodeBase64Utf8(encoded: string): string {
    return new TextDecoder().decode(Uint8Array.from(atob(encoded), (character) => character.charCodeAt(0)));
}

// parseBase64Json decodes base64-encoded UTF-8 text and parses it as JSON. Throws on malformed base64
// or invalid JSON, as JSON.parse would.
export function parseBase64Json<T = any>(encoded: string): T {
    return JSON.parse(decodeBase64Utf8(encoded));
}
