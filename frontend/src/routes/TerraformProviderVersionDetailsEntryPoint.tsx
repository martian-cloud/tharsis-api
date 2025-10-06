import { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import TerraformProviderDetails from '../providers/TerraformProviderVersionDetails';
import TerraformProviderVersionDetailsQuery, { TerraformProviderVersionDetailsQuery as TerraformProviderVersionDetailsQueryType } from "../providers/__generated__/TerraformProviderVersionDetailsQuery.graphql";

function TerraformProviderVersionDetailsEntryPoint() {
    const { registryNamespace, providerName, version } = useParams();

    const [queryRef, loadQuery] = useQueryLoader<TerraformProviderVersionDetailsQueryType>(TerraformProviderVersionDetailsQuery)

    useEffect(() => {
        loadQuery(
            { registryNamespace: registryNamespace as string, providerName: providerName as string, version: version || null },
            { fetchPolicy: 'store-and-network' }
        );
    }, [loadQuery, registryNamespace, providerName, version])

    return queryRef != null ? <TerraformProviderDetails queryRef={queryRef} /> : null
}

export default TerraformProviderVersionDetailsEntryPoint;
