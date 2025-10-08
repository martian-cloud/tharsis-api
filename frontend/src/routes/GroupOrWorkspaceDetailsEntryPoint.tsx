import { useEffect, useState } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import GroupOrWorkspaceRenderer from './GroupOrWorkspaceRenderer';
import GroupOrWorkspaceRendererQuery, { GroupOrWorkspaceRendererQuery as GroupOrWorkspaceRendererQueryType } from "./__generated__/GroupOrWorkspaceRendererQuery.graphql";

function parseParams(path: string): { path: string, route: string } {
    let route = '';

    // Check if path contains hyphen
    if (path.includes('/-/')) {
        const parts = path.split('/');
        const index = parts.indexOf('-');
        path = parts.slice(0, index).join('/')
        if (parts.length > (index + 1)) {
            route = parts.slice(index + 1)[0]
        }
    }

    return { path, route };
}

function GroupOrWorkspaceDetailsEntryPoint() {
    const params = useParams();
    const splat = params['*'] as string;
    const [queryRef, loadQuery] = useQueryLoader<GroupOrWorkspaceRendererQueryType>(GroupOrWorkspaceRendererQuery);
    const [fullPath, setFullPath] = useState('')
    const [route, setRoute] = useState('')

    useEffect(() => {
        const resp = parseParams(splat);
        setFullPath(resp.path);
        setRoute(resp.route);
    }, [splat]);

    useEffect(() => {
        if (fullPath) {
            loadQuery({ fullPath: fullPath }, { fetchPolicy: 'store-and-network' })
        }
    }, [loadQuery, fullPath]);

    return queryRef != null ? <GroupOrWorkspaceRenderer queryRef={queryRef} route={route} /> : null
}

export default GroupOrWorkspaceDetailsEntryPoint;
