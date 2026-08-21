import { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import PackageSearch, { INITIAL_ITEM_COUNT } from '../packages/PackageSearch';
import PackageSearchQuery, { PackageSearchQuery as PackageSearchQueryType } from "../packages/__generated__/PackageSearchQuery.graphql";

function PackageSearchEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<PackageSearchQueryType>(PackageSearchQuery);
    const [searchParams] = useSearchParams();

    const search = searchParams.get('search') || '';

    useEffect(() => {
        loadQuery({
            first: INITIAL_ITEM_COUNT,
            search: search,
        }, { fetchPolicy: 'store-and-network' });
    }, [loadQuery, search]);

    return queryRef != null ? (
        <PackageSearch
            queryRef={queryRef}
            search={search}
        />
    ) : null;
}

export default PackageSearchEntryPoint;
