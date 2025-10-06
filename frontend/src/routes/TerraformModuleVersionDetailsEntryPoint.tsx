import { useEffect } from 'react';
import { useQueryLoader } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import TerraformModuleVersionDetails from '../modules/TerraformModuleVersionDetails';
import { INITIAL_ITEM_COUNT } from '../modules/TerraformModuleVersionAttestList';
import TerraformModuleVersionDetailsQuery, { TerraformModuleVersionDetailsQuery as TerraformModuleVersionDetailsQueryType } from "../modules/__generated__/TerraformModuleVersionDetailsQuery.graphql";

function TerraformModuleVersionDetailsEntryPoint() {
    const { registryNamespace, moduleName, system, version } = useParams();

    const [queryRef, loadQuery] = useQueryLoader<TerraformModuleVersionDetailsQueryType>(TerraformModuleVersionDetailsQuery)

    useEffect(() => {
        loadQuery(
            { registryNamespace: registryNamespace as string, moduleName: moduleName as string, system: system as string, version: version || null, first: INITIAL_ITEM_COUNT },
            { fetchPolicy: 'store-and-network' }
        );
    }, [loadQuery, registryNamespace, moduleName, version])

    return queryRef != null ? <TerraformModuleVersionDetails queryRef={queryRef} /> : null
}

export default TerraformModuleVersionDetailsEntryPoint;
