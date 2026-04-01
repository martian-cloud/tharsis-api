import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import TableCell from '@mui/material/TableCell';
import TableRow from '@mui/material/TableRow';
import graphql from 'babel-plugin-relay/macro';
import React, { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import AmazonwebservicesPlainWordmarkIcon from 'react-devicons/amazonwebservices/plain-wordmark';
import KubernetesPlainIcon from 'react-devicons/kubernetes/plain';
import HelmPlainIcon from 'react-devicons/helm/original';
import TerraformPlainIcon from 'react-devicons/terraform/plain';
import AzurePlainIcon from 'react-devicons/azure/plain';
import { TharsisIcon } from '../../common/Icons';
import { SvgIconProps } from '@mui/material/SvgIcon';
import { StateVersionResourceListItemFragment_resource$key } from './__generated__/StateVersionResourceListItemFragment_resource.graphql';

// Adapts TharsisIcon to accept the `size` prop used by react-devicons icons
function TharsisIconAdapter({ size, ...rest }: { size?: number } & SvgIconProps) {
    return <TharsisIcon sx={{ fontSize: size }} {...rest} />;
}

const PROVIDER_ICONS: Record<string, React.ComponentType<any>> = {
    aws: AmazonwebservicesPlainWordmarkIcon,
    kubernetes: KubernetesPlainIcon,
    helm: HelmPlainIcon,
    terraform: TerraformPlainIcon,
    azurerm: AzurePlainIcon,
    tharsis: TharsisIconAdapter,
};

function getProviderIcon(provider: string | undefined): React.ComponentType<any> {
    if (!provider) return TerraformPlainIcon;
    const shortName = (provider.split('/').pop() || '').toLowerCase();
    return PROVIDER_ICONS[shortName] || TerraformPlainIcon;
}

interface Props {
    fragmentRef: StateVersionResourceListItemFragment_resource$key
}

function StateVersionResourceListItem(props: Props) {
    const { fragmentRef } = props;
    const data = useFragment<StateVersionResourceListItemFragment_resource$key>(
        graphql`
        fragment StateVersionResourceListItemFragment_resource on StateVersionResource
        {
            name
            type
            provider
            mode
            module
        }
      `, fragmentRef);

    const ProviderIcon = useMemo(() => getProviderIcon(data.provider), [data.provider]);

    return (
        <TableRow
            sx={{ '&:last-child td, &:last-child th': { border: 0 } }}
        >
            <TableCell sx={{ wordBreak: 'break-all' }}>
                {data.name}
            </TableCell>
            <TableCell sx={{ wordBreak: 'break-all' }}>
                {data.type}
                {data.mode === 'data' && <Chip size="small" label='datasource' sx={{ marginLeft: 1}} />}
            </TableCell>
            <TableCell sx={{ wordBreak: 'break-all' }}>
                <Box display="flex" alignItems="center" gap={1}>
                    <ProviderIcon size={20} color="currentColor" />
                    {data.provider}
                </Box>
            </TableCell>
            <TableCell sx={{ wordBreak: 'break-all' }}>
                {data.module}
            </TableCell>
        </TableRow>
    );
}

export default StateVersionResourceListItem;