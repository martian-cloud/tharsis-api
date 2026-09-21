import EmailRecipientClickTracker from './email/EmailRecipientClickTracker';

// Mounts every component that fires a one-time action from a query param on load (see
// useQueryParamAction), so adding a new one only means editing this file rather than Root.tsx.
function GlobalQueryParamActions() {
    return (
        <>
            <EmailRecipientClickTracker />
        </>
    );
}

export default GlobalQueryParamActions;
