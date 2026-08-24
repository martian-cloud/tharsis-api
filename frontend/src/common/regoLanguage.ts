// Rego (Open Policy Agent) language support for Monaco.
//
// Monaco has no built-in Rego language, so we register a custom `rego` language
// with a Monarch tokenizer (syntax highlighting) and a language configuration
// (comment toggling, bracket/quote auto-close). This is highlighting only — not
// completions or semantic validation. Registered against our self-hosted Monaco
// instance from src/common/monaco.ts.
import type * as monacoNs from 'monaco-editor';

const REGO_LANGUAGE_ID = 'rego';

// Reserved words per the OPA Rego grammar.
const keywords = [
    'package', 'import', 'as', 'default', 'else', 'not',
    'some', 'every', 'in', 'with', 'if', 'contains',
];
const constants = ['true', 'false', 'null'];
// Document roots — highlighted distinctly from ordinary identifiers.
const builtins = ['input', 'data'];
const operators = [
    ':=', '==', '!=', '<=', '>=', '=', '<', '>',
    '+', '-', '*', '/', '%', '&', '|',
];

const tokenizer: monacoNs.languages.IMonarchLanguage = {
    defaultToken: '',
    keywords,
    constants,
    builtins,
    operators,
    symbols: /[=><!~?:&|+\-*/%]+/,
    // Rego string escapes are JSON's, not Go's: the OPA scanner accepts only \\ \" \/ \b \f \n \r \t
    // and \uXXXX, and errors on anything else. Kept narrow so the editor does not colour an escape as
    // valid that the API will reject when the policy is saved.
    escapes: /\\(?:["\\/bfnrt]|u[0-9A-Fa-f]{4})/,
    tokenizer: {
        root: [
            [/[a-zA-Z_]\w*/, {
                cases: {
                    '@keywords': 'keyword',
                    '@constants': 'constant',
                    '@builtins': 'variable.predefined',
                    '@default': 'identifier',
                },
            }],
            { include: '@whitespace' },
            [/[{}()[\]]/, '@brackets'],
            [/@symbols/, { cases: { '@operators': 'operator', '@default': '' } }],
            [/\d+\.\d+(?:[eE][-+]?\d+)?/, 'number.float'],
            [/\d+/, 'number'],
            [/"/, { token: 'string.quote', bracket: '@open', next: '@string' }],
            [/`/, { token: 'string.quote', bracket: '@open', next: '@rawstring' }],
            [/[;,.]/, 'delimiter'],
        ],
        whitespace: [
            [/[ \t\r\n]+/, ''],
            [/#.*$/, 'comment'],
        ],
        string: [
            [/[^\\"]+/, 'string'],
            [/@escapes/, 'string.escape'],
            [/\\./, 'string.escape.invalid'],
            [/"/, { token: 'string.quote', bracket: '@close', next: '@pop' }],
        ],
        rawstring: [
            [/[^`]+/, 'string'],
            [/`/, { token: 'string.quote', bracket: '@close', next: '@pop' }],
        ],
    },
};

const languageConfiguration: monacoNs.languages.LanguageConfiguration = {
    comments: { lineComment: '#' },
    brackets: [
        ['{', '}'],
        ['[', ']'],
        ['(', ')'],
    ],
    autoClosingPairs: [
        { open: '{', close: '}' },
        { open: '[', close: ']' },
        { open: '(', close: ')' },
        { open: '"', close: '"', notIn: ['string'] },
        { open: '`', close: '`', notIn: ['string'] },
    ],
    surroundingPairs: [
        { open: '{', close: '}' },
        { open: '[', close: ']' },
        { open: '(', close: ')' },
        { open: '"', close: '"' },
        { open: '`', close: '`' },
    ],
};

// registerRegoLanguage registers the `rego` language on the given Monaco
// instance. Idempotent — safe to call more than once.
export function registerRegoLanguage(m: typeof monacoNs): void {
    if (m.languages.getLanguages().some(lang => lang.id === REGO_LANGUAGE_ID)) {
        return;
    }
    m.languages.register({ id: REGO_LANGUAGE_ID, extensions: ['.rego'], aliases: ['Rego', 'rego'] });
    m.languages.setMonarchTokensProvider(REGO_LANGUAGE_ID, tokenizer);
    m.languages.setLanguageConfiguration(REGO_LANGUAGE_ID, languageConfiguration);
}
