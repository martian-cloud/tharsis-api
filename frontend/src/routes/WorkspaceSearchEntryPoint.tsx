import React, { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import WorkspaceSearch, { INITIAL_ITEM_COUNT } from '../workspace/WorkspaceSearch';
import WorkspaceSearchQuery, { WorkspaceSearchQuery as WorkspaceSearchQueryType } from "../workspace/__generated__/WorkspaceSearchQuery.graphql";

function WorkspaceSearchEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<WorkspaceSearchQueryType>(WorkspaceSearchQuery);
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
    }, [loadQuery]);

    return queryRef != null ? (
        <WorkspaceSearch
            queryRef={queryRef}
            search={search}
            labelFilters={labelFilters}
            filterExpanded={filterExpanded}
        />
    ) : null;
}

export default WorkspaceSearchEntryPoint;
