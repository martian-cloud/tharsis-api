import { Box, Tooltip, Typography } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useFragment, useLazyLoadQuery } from 'react-relay';
import { useParams } from 'react-router-dom';
import Gravatar from '../../common/Gravatar';
import Timestamp from '../../common/Timestamp';
import TRNButton from '../../common/TRNButton';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import StateVersionFile from './StateVersionFile';
import { StateVersionDetailsFragment_details$key } from './__generated__/StateVersionDetailsFragment_details.graphql';
import { StateVersionDetailsQuery } from './__generated__/StateVersionDetailsQuery.graphql';

interface Props {
    fragmentRef: StateVersionDetailsFragment_details$key
}

function StateVersionDetails(props: Props) {
    const params = useParams();
    const stateVersionId = params.id as string;

    const data = useFragment<StateVersionDetailsFragment_details$key>(
        graphql`
        fragment StateVersionDetailsFragment_details on Workspace
        {
            id
            fullPath
        }
    `, props.fragmentRef)

    const queryData = useLazyLoadQuery<StateVersionDetailsQuery>(graphql`
        query StateVersionDetailsQuery($id: String!) {
            node(id: $id) {
                ... on StateVersion {
                    id
                    createdBy
                    metadata {
                        createdAt
                        trn
                    }
                    run {
                        createdBy
                    }
                    ...StateVersionFileFragment_stateVersion
                }
            }
        }
    `, { id: stateVersionId }, { fetchPolicy: 'store-and-network' })

    const createdBy = queryData.node?.run ? queryData.node.run.createdBy : (queryData.node?.createdBy || '')

    return queryData.node ? (
        <Box>
            <NamespaceBreadcrumbs
                namespacePath={data.fullPath}
                childRoutes={[
                    { title: "state versions", path: 'state_versions' },
                    { title: `${stateVersionId.substring(0, 8)}...`, path: stateVersionId }
                ]}
            />
            <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, justifyContent: 'space-between', alignItems: { xs: 'flex-start', sm: 'center' }, gap: 1 }}>
                <Typography component="div" sx={{ paddingRight: '6px' }}>State version created{' '}
                    <Timestamp timestamp={queryData.node?.metadata?.createdAt as string} />
                    {' '}by
                    <Tooltip title={createdBy}>
                        <Box sx={{ display: 'inline-flex', verticalAlign: 'middle', ml: 1 }}>
                            <Gravatar width={20} height={20} email={createdBy} />
                        </Box>
                    </Tooltip>
                </Typography>
                <TRNButton trn={queryData.node?.metadata?.trn || ''} size="small" />
            </Box>
            <StateVersionFile fragmentRef={queryData.node} />
        </Box>
    ) : <Box>Not Found</Box>;
}

export default StateVersionDetails
