import { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import TeamDetails from '../teams/TeamDetails';
import TeamDetailsQuery, { TeamDetailsQuery as TeamDetailsQueryType } from '../teams/__generated__/TeamDetailsQuery.graphql';

function TeamDetailsEntryPoint() {
    const [queryRef, loadQuery] = useQueryLoader<TeamDetailsQueryType>(TeamDetailsQuery);
    const teamName = useParams().teamName as string;

    useEffect(() => {
        loadQuery({ name: teamName, first: 20 }, { fetchPolicy: 'store-and-network' });
    }, [loadQuery]);

    return queryRef != null ? <TeamDetails queryRef={queryRef} /> : null;
}

export default TeamDetailsEntryPoint;
