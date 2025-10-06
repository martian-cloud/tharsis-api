function buildGraphqlUrl(protocol: 'http' | 'ws'): string {
    const urlSetting = import.meta.env.VITE_THARSIS_API_URL ? import.meta.env.VITE_THARSIS_API_URL : (window as any).env.THARSIS_API_URL;

    let ssl: boolean;
    let host: string;

    if (urlSetting && urlSetting !== '__THARSIS_API_URL__') {
        const apiUrl = new URL(urlSetting);
        ssl = apiUrl.protocol === 'https:';
        host = apiUrl.host;
    } else {
        // Use current page's host when UI runs embedded with API
        ssl = window.location.protocol === 'https:';
        host = window.location.host;
    }

    const scheme = protocol === 'http' ? (ssl ? 'https' : 'http') : (ssl ? 'wss' : 'ws');
    return `${scheme}://${host}`;
}

const cfg = {
    apiUrl: buildGraphqlUrl('http'),
    wsUrl: buildGraphqlUrl('ws'),
    docsUrl: 'https://tharsis.martian-cloud.io',
};

export default cfg;
