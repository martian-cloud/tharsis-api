import React, { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import Home from '../home/Home';
import HomeQuery, { HomeQuery as HomeQueryType } from '../home/__generated__/HomeQuery.graphql';

function HomeEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<HomeQueryType>(HomeQuery)

    useEffect(() => {
        loadQuery({}, { fetchPolicy: 'store-and-network' })
    }, [loadQuery])

    return queryRef != null ? <Home queryRef={queryRef} /> : null
}

export default HomeEntryPoint;
