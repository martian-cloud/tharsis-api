import { VCSProviderOption } from './VCSProviderAutocomplete';

export function providerLabel(providerType: string | undefined): string {
    return providerType === 'gitlab' ? 'GitLab' : 'GitHub';
}

// manualWebhookDetailsLabel names what a manual webhook needs pasted in: GitHub carries the token in the URL.
export function manualWebhookDetailsLabel(providerType: string | undefined): string {
    return providerType === 'gitlab' ? 'URL and token' : 'URL';
}

// providerHelperText describes what happens to the webhook once a link uses the chosen provider.
export function providerHelperText(provider: VCSProviderOption | null, fallback: string): string {
    if (!provider) {
        return fallback;
    }

    return provider.autoCreateWebhooks
        ? 'Tharsis creates the webhook in the repository for you.'
        : `You will copy a webhook ${manualWebhookDetailsLabel(provider.type)} into ${providerLabel(provider.type)} after linking.`;
}
