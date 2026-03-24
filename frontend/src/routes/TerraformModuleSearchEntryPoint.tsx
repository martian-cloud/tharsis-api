import React, { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import TerraformModuleSearch, { INITIAL_ITEM_COUNT } from '../modules/TerraformModuleSearch';
import TerraformModuleSearchQuery, { TerraformModuleSearchQuery as TerraformModuleSearchQueryType } from "../modules/__generated__/TerraformModuleSearchQuery.graphql";

function TerraformModuleSearchEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<TerraformModuleSearchQueryType>(TerraformModuleSearchQuery);
    const [searchParams] = useSearchParams();

    const search = searchParams.get('search') || '';
    const filterExpanded = searchParams.get('filterExpanded') === 'true';
    const labelFilters = React.useMemo(() => {
        const filters: Array<{ key: string; value: string }> = [];

        // Parse label filters from URL params in format: label.key=value
        searchParams.forEach((value, key) => {
            if (key.startsWith('label.')) {
                const labelKey = key.substring(6); // Remove 'label.' prefix
                if (labelKey && value) {
                    filters.push({ key: labelKey, value });
                }
            }
        });

        return filters;
    }, [searchParams]);

    useEffect(() => {
        loadQuery({
            first: INITIAL_ITEM_COUNT,
            search: search,
            labelFilter: {
                labels: labelFilters,
            },
        }, { fetchPolicy: 'store-and-network' });
    }, [loadQuery, search, labelFilters]);

    return queryRef != null ? (
        <TerraformModuleSearch
            queryRef={queryRef}
            search={search}
            labelFilters={labelFilters}
            filterExpanded={filterExpanded}
        />
    ) : null;
}

export default TerraformModuleSearchEntryPoint;
