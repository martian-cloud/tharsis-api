import React, { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import WorkspaceSearch, { INITIAL_ITEM_COUNT } from '../workspace/WorkspaceSearch';
import WorkspaceSearchQuery, { WorkspaceSearchQuery as WorkspaceSearchQueryType } from "../workspace/__generated__/WorkspaceSearchQuery.graphql";

function WorkspaceSearchEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<WorkspaceSearchQueryType>(WorkspaceSearchQuery)

    useEffect(() => {
        loadQuery({ first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' })
    }, [loadQuery])

    return queryRef != null ? <WorkspaceSearch queryRef={queryRef} /> : null
}

export default WorkspaceSearchEntryPoint;
