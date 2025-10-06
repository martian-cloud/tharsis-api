import React, { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import TerraformModuleSearch, { INITIAL_ITEM_COUNT } from '../modules/TerraformModuleSearch';
import TerraformModuleSearchQuery, { TerraformModuleSearchQuery as TerraformModuleSearchQueryType } from "../modules/__generated__/TerraformModuleSearchQuery.graphql";

function TerraformModuleSearchEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<TerraformModuleSearchQueryType>(TerraformModuleSearchQuery)

    useEffect(() => {
        loadQuery({ first: INITIAL_ITEM_COUNT }, { fetchPolicy: 'store-and-network' })
    }, [loadQuery])

    return queryRef != null ? <TerraformModuleSearch queryRef={queryRef} /> : null
}

export default TerraformModuleSearchEntryPoint;
