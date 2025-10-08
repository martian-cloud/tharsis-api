import { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import UserPreferences from '../userpreferences/UserPreferences';
import UserPreferencesQuery, { UserPreferencesQuery as UserPreferencesQueryType } from '../userpreferences/__generated__/UserPreferencesQuery.graphql';

function UserPreferencesEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<UserPreferencesQueryType>(UserPreferencesQuery)

    useEffect(() => {
        loadQuery({ first: 10 }, { fetchPolicy: 'store-and-network' })
    }, [loadQuery])

    return queryRef != null ? <UserPreferences queryRef={queryRef} /> : null
}

export default UserPreferencesEntryPoint;
