import graphql from 'babel-plugin-relay/macro';
import { useCallback } from 'react';
import { useMutation } from 'react-relay/hooks';
import useQueryParamAction from '../common/useQueryParamAction';
import { EmailRecipientClickTrackerMutation } from './__generated__/EmailRecipientClickTrackerMutation.graphql';

// Emails link back into the app with a `?rid=<recipientId>` query param on the recipient's own click,
// so a recipient who blocks images (and never triggers the provider's open-tracking pixel) still
// registers as clicked once they land on an authenticated page.
const RECIPIENT_ID_PARAM = 'rid';

function EmailRecipientClickTracker() {
    const [commit] = useMutation<EmailRecipientClickTrackerMutation>(graphql`
        mutation EmailRecipientClickTrackerMutation($input: MarkEmailRecipientClickedInput!) {
            markEmailRecipientClicked(input: $input) {
                emailRecipient {
                    id
                }
                problems {
                    message
                    type
                }
            }
        }
    `);

    const markClicked = useCallback((recipientId: string) => {
        // Best-effort: a failure here shouldn't bother the user, but should still be visible for debugging.
        commit({
            variables: { input: { recipientId } },
            onError: (error) => console.error('Failed to mark email recipient clicked', error),
        });
    }, [commit]);

    useQueryParamAction(RECIPIENT_ID_PARAM, markClicked);

    return null;
}

export default EmailRecipientClickTracker;
