import { Typography } from '@mui/material'
import Box from '@mui/material/Box'
import Link from '../../routes/Link'
import ActorDisplay from '../../common/ActorDisplay'
import { ResponsiveRow } from '../../common/ResponsiveTable'
import Timestamp from '../../common/Timestamp'
import { useFragment } from 'react-relay/hooks'
import { StateVersionListItemFragment_stateVersion$key } from './__generated__/StateVersionListItemFragment_stateVersion.graphql'
import graphql from 'babel-plugin-relay/macro'

interface Props {
    stateVersionKey: StateVersionListItemFragment_stateVersion$key
    workspacePath: string
}

function StateVersionListItem(props: Props) {
    const { stateVersionKey, workspacePath } = props

    const data = useFragment<StateVersionListItemFragment_stateVersion$key>(graphql`
        fragment StateVersionListItemFragment_stateVersion on StateVersion {
            id
            createdByActor {
                ...ActorDisplayFragment_actor
            }
            metadata {
                createdAt
                trn
            }
            run {
                id
                createdByActor {
                    ...ActorDisplayFragment_actor
                }
            }
        } `, stateVersionKey)

    const stateVersionActor = data.run ? data.run.createdByActor : data.createdByActor

    return (
        <ResponsiveRow cells={[
            { primary: true, content: <Link color="inherit" to={`/groups/${workspacePath}/-/state_versions/${data.id}`}>{data.id.substring(0, 8)}...</Link> },
            { label: 'Run ID', content: data.run ? <Link color="inherit" to={`/groups/${workspacePath}/-/runs/${data.run.id}`}>{data.run.id.substring(0, 8)}...</Link> : <Typography variant="body2" color="textSecondary">created manually</Typography> },
            { label: 'Created At', content: <Timestamp variant="body2" timestamp={data.metadata.createdAt} /> },
            {
                label: 'Created By', content: (
                    <Box sx={{ display: 'flex', width: 'fit-content' }}>
                        <ActorDisplay fragmentRef={stateVersionActor} variant="avatar" size={24} />
                    </Box>
                )
            },
        ]} />
    );
}

export default StateVersionListItem
