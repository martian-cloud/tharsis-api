import { tokenize } from 'react-diff-view';
import refractor from 'refractor';

// Add event listener for tokenize web worker
self.addEventListener(
    'message',
    ({ data: { id, payload } }) => {
        const { hunks, oldSource, language, enhancers } = payload;

        const options = {
            highlight: language !== 'text',
            refractor: refractor,
            language: language,
            oldSource: oldSource,
            enhancers: enhancers,
        };

        try {
            const tokens = tokenize(hunks, options);
            const payload = {
                success: true,
                tokens: tokens,
            };
            self.postMessage({ id, payload });
        }
        catch (ex) {
            const payload = {
                success: false,
                reason: ex instanceof Error ? ex.message : `${ex}`,
            };
            self.postMessage({ id, payload });
        }
    }
);
