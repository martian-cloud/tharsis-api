import { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import GroupTreeContainer, { DEFAULT_SORT, INITIAL_ITEM_COUNT } from '../groups/tree/GroupTreeContainer';
import GroupTreeContainerQuery, { GroupTreeContainerQuery as GroupTreeContainerQueryType } from '../groups/tree/__generated__/GroupTreeContainerQuery.graphql';

function ExploreGroupsEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<GroupTreeContainerQueryType>(GroupTreeContainerQuery)

    useEffect(() => {
        loadQuery({ first: INITIAL_ITEM_COUNT, parentPath: '', sort: DEFAULT_SORT }, { fetchPolicy: 'store-and-network' })
    }, [loadQuery])

    return queryRef != null ? <GroupTreeContainer queryRef={queryRef} /> : null
}

export default ExploreGroupsEntryPoint;
