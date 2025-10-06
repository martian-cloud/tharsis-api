import React, { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import TerraformProviderSearch, { INITIAL_ITEM_COUNT } from '../providers/TerraformProviderSearch';
import TerraformProviderSearchQuery, { TerraformProviderSearchQuery as TerraformProviderSearchQueryType } from "../providers/__generated__/TerraformProviderSearchQuery.graphql";

function TerraformProviderSearchEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<TerraformProviderSearchQueryType>(TerraformProviderSearchQuery)

    useEffect(() => {
        loadQuery({ first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' })
    }, [loadQuery])

    return queryRef != null ? <TerraformProviderSearch queryRef={queryRef} /> : null
}

export default TerraformProviderSearchEntryPoint;
