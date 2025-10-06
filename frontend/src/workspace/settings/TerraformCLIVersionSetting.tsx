import { Box, MenuItem, Select, Typography } from '@mui/material'
import React from 'react'
import graphql from 'babel-plugin-relay/macro'
import { useLazyLoadQuery } from 'react-relay/hooks';
import { TerraformCLIVersionSettingQuery } from './__generated__/TerraformCLIVersionSettingQuery.graphql'

interface Props {
    data: string
    onChange: (event: any) => void
}

function TerraformCLIVersionSetting(props: Props) {
    const { data, onChange } = props

    const versionsData = useLazyLoadQuery<TerraformCLIVersionSettingQuery>(graphql`
        query TerraformCLIVersionSettingQuery {
            terraformCLIVersions {
                versions
            }
        }`, {}, { fetchPolicy: 'store-or-network' })

    return (
        <Box sx={{ mb: 4 }}>
            <Typography variant="subtitle1" gutterBottom>Terraform CLI Version</Typography>
            <Box>
                <Select
                    sx={{ width: 150, mb: 1 }}
                    size="small"
                    labelId="terraform-cli-versions-select-label"
                    id="terraform-cli-versions-select"
                    value={data}
                    onChange={event => onChange(event)}
                >
                    {versionsData.terraformCLIVersions ? [...versionsData.terraformCLIVersions.versions].reverse().map((opt: string) => <MenuItem key={opt} value={opt}>{opt}</MenuItem>) : null}
                </Select>
                <Typography variant="subtitle2">The latest version was selected when the workspace was created. The version will <strong>not upgrade automatically</strong> to the latest version. Version changes must be made manually.</Typography>
            </Box>
        </Box>
    )
}

export default TerraformCLIVersionSetting
