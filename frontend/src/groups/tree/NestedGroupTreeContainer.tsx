import graphql from 'babel-plugin-relay/macro';
import React from 'react';
import { useLazyLoadQuery, usePaginationFragment } from 'react-relay/hooks';
import GroupTree from './GroupTree';
import { NestedGroupTreeContainerFragment_groups$key } from './__generated__/NestedGroupTreeContainerFragment_groups.graphql';
import { NestedGroupTreeContainerQuery } from './__generated__/NestedGroupTreeContainerQuery.graphql';
import { NestedGroupsListPaginationQuery } from './__generated__/NestedGroupsListPaginationQuery.graphql';

interface Props {
    parentPath: string
}

function NestedGroupTreeContainer(props: Props) {
    const queryData = useLazyLoadQuery<NestedGroupTreeContainerQuery>(graphql`
        query NestedGroupTreeContainerQuery($first: Int, $last: Int, $after: String, $before: String, $parentPath: String!) {
            ...NestedGroupTreeContainerFragment_groups
        }
    `, { first: 100, parentPath: props.parentPath })

    const { data, loadNext, hasNext, isLoadingNext } = usePaginationFragment<NestedGroupsListPaginationQuery, NestedGroupTreeContainerFragment_groups$key>(graphql`
        fragment NestedGroupTreeContainerFragment_groups on Query
        @refetchable(queryName: "NestedGroupsListPaginationQuery") {
            groups(
                after: $after
                before: $before
                first: $first
                last: $last
                parentPath: $parentPath
                sort:FULL_PATH_ASC
            ) @connection(key: "NestedGroupTreeContainer_groups") {
                edges {
                    node {
                        id
                    }
                }
                ...GroupTreeFragment_connection
            }
        }
    `, queryData)

    return <GroupTree connectionKey={data.groups} nested loadNext={loadNext} hasNext={hasNext} isLoadingNext={isLoadingNext} />;
}

export default NestedGroupTreeContainer
