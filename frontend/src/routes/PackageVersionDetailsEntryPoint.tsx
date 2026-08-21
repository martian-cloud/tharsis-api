import { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import RegistryPackageVersionDetails from '../packages/RegistryPackageVersionDetails';
import RegistryPackageVersionDetailsQuery, { RegistryPackageVersionDetailsQuery as RegistryPackageVersionDetailsQueryType } from "../packages/__generated__/RegistryPackageVersionDetailsQuery.graphql";

const INITIAL_ITEM_COUNT = 20;

function PackageVersionDetailsEntryPoint() {
    const { packageId } = useParams();

    const [queryRef, loadQuery] = useQueryLoader<RegistryPackageVersionDetailsQueryType>(RegistryPackageVersionDetailsQuery);

    useEffect(() => {
        loadQuery(
            { id: packageId as string, first: INITIAL_ITEM_COUNT },
            { fetchPolicy: 'store-and-network' }
        );
    }, [loadQuery, packageId]);

    return queryRef != null ? <RegistryPackageVersionDetails queryRef={queryRef} /> : null;
}

export default PackageVersionDetailsEntryPoint;
