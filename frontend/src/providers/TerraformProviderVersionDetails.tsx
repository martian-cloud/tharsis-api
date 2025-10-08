import CopyIcon from '@mui/icons-material/ContentCopy';
import DoubleArrowIcon from '@mui/icons-material/DoubleArrow';
import { Alert, Chip, CircularProgress, IconButton, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import { useTheme } from '@mui/material/styles';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import useMediaQuery from '@mui/material/useMediaQuery';
import graphql from 'babel-plugin-relay/macro';
import React, { Suspense, useState } from 'react';
import { PreloadedQuery, useFragment, usePreloadedQuery } from 'react-relay/hooks';
import { useSearchParams, useParams } from 'react-router-dom';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { atomDark as prismTheme } from 'react-syntax-highlighter/dist/esm/styles/prism';
import remarkGfm from 'remark-gfm';
import MuiMarkdown from '../common/Markdown';
import TRNButton from '../common/TRNButton';
import ListSkeleton from '../skeletons/ListSkeleton';
import TerraformProviderVersionDetailsSidebar, { SidebarWidth } from './TerraformProviderVersionDetailsSidebar';
import TerraformProviderVersionList from './TerraformProviderVersionList';
import { TerraformProviderVersionDetailsIndexFragment_details$key } from './__generated__/TerraformProviderVersionDetailsIndexFragment_details.graphql';
import { TerraformProviderVersionDetailsQuery } from './__generated__/TerraformProviderVersionDetailsQuery.graphql';

const query = graphql`
    query TerraformProviderVersionDetailsQuery($registryNamespace: String!, $providerName: String!, $version: String) {
      terraformProviderVersion(registryNamespace: $registryNamespace, providerName: $providerName, version: $version) {
        id
        ...TerraformProviderVersionDetailsIndexFragment_details
      }
    }
`;

function buildUsageInfo(registryNamespace: string, providerName: string, version: string) {
  return `terraform {
 required_providers {
    ${providerName} = {
       source  = "${window.location.host}/${registryNamespace}/${providerName}"
       version = "${version}"
    }
 }
}

provider "${providerName}" {
 # Configuration options
}`;
}

interface Props {
  queryRef: PreloadedQuery<TerraformProviderVersionDetailsQuery>
}

function TerraformProviderVersionDetails(props: Props) {
  const { queryRef } = props;
  const { registryNamespace, providerName, version } = useParams();
  const queryData = usePreloadedQuery<TerraformProviderVersionDetailsQuery>(query, queryRef);

  return (
    <Box display="flex">
      <Box component="main" flexGrow={1}>
        <Suspense fallback={<Box
          sx={{
            width: '100%',
            height: `calc(100vh - 64px)`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center'
          }}
        >
          <CircularProgress />
        </Box>}>
          <Box maxWidth={1400} margin="auto" padding={2}>
            {queryData.terraformProviderVersion && <TerraformProviderVersionDetailsIndex fragmentRef={queryData.terraformProviderVersion} />}
            {!queryData.terraformProviderVersion && <Box display="flex" justifyContent="center" marginTop={4}>
              <Typography variant="h6" color="textSecondary">
                version <strong>{version || 'latest'}</strong> not found for provider <strong>{registryNamespace}/{providerName}</strong>
              </Typography>
            </Box>}
          </Box>
        </Suspense>
      </Box>
    </Box>
  );
}

interface IndexProps {
  fragmentRef: TerraformProviderVersionDetailsIndexFragment_details$key
}

function TerraformProviderVersionDetailsIndex(props: IndexProps) {
  const [searchParams, setSearchParams] = useSearchParams();

  const theme = useTheme();
  const mobile = useMediaQuery(theme.breakpoints.down('md'));

  const [sidebarOpen, setSidebarOpen] = useState(false);

  const data = useFragment<TerraformProviderVersionDetailsIndexFragment_details$key>(
    graphql`
          fragment TerraformProviderVersionDetailsIndexFragment_details on TerraformProviderVersion
          {
              id
              version
              readme
              shaSumsUploaded
              shaSumsSigUploaded
              metadata {
                  trn
              }
              provider {
                  id
                  name
                  registryNamespace
                  private
                  ...TerraformProviderVersionListFragment_provider
              }
              ...TerraformProviderVersionDetailsSidebarFragment_details
          }
        `, props.fragmentRef);

  let tab = searchParams.get('tab') || 'readme';
  if (data.readme === '' && tab === 'readme') {
    tab = 'usage'
  }

  const onToggleSidebar = () => {
    setSidebarOpen(prev => !prev);
  };

  const onTabChange = (event: React.SyntheticEvent, newValue: string) => {
    searchParams.set('tab', newValue);
    setSearchParams(searchParams, { replace: true });
  };

  const usageInfo = buildUsageInfo(data.provider.registryNamespace, data.provider.name, data.version);

  return (
    <Box>
      <TerraformProviderVersionDetailsSidebar
        fragmentRef={data}
        open={sidebarOpen}
        temporary={mobile}
        onClose={onToggleSidebar}
      />
      <Box>
        <Box paddingRight={!mobile ? `${SidebarWidth}px` : 0}>
          {(!data.shaSumsUploaded || !data.shaSumsSigUploaded) && <Alert sx={{ marginBottom: 2 }} severity="warning">
            This provider version is missing the required checksum files
          </Alert>}
          <Box
            sx={{
              display: 'flex',
              marginBottom: 2,
              justifyContent: 'space-between',
              flexDirection: { xs: 'column', md: 'row' },
              alignItems: { xs: 'flex-start', md: 'center' },
              gap: { xs: 2 }
            }}
          >
            <Box display="flex" alignItems="center" justifyContent="space-between" width="100%">
              <Box display="flex" alignItems="center">
                <Typography variant="h6">{data.provider.registryNamespace} / {data.provider.name}</Typography>
                {data.provider.private && <Chip sx={{ marginLeft: 2 }} variant="outlined" color="warning" size="small" label="private" />}
              </Box>
              <IconButton
                onClick={onToggleSidebar}
                sx={{ display: { xs: 'block', md: 'none' } }}
              >
                <DoubleArrowIcon sx={{ transform: 'rotate(180deg)' }} />
              </IconButton>
            </Box>
            <Box>
              <TRNButton trn={data.metadata.trn} size="small" />
            </Box>
          </Box>
          <Box sx={{ border: 1, borderColor: 'divider', marginBottom: 2 }}>
            <Tabs value={tab} onChange={onTabChange}>
              {data.readme !== "" && <Tab label="README" value="readme" />}
              <Tab label="How To Use" value="usage" />
              <Tab label="Versions" value="versions" />
            </Tabs>
          </Box>
          <React.Fragment>
            {tab === 'readme' && <MuiMarkdown
              children={data.readme}
              remarkPlugins={[remarkGfm]}
            />}
            {tab === 'usage' && <Box marginTop={2} position="relative">
              <IconButton sx={{ padding: 2, position: 'absolute', top: 0, right: 0 }} onClick={() => navigator.clipboard.writeText(usageInfo)}>
                <CopyIcon sx={{ width: 16, height: 16 }} />
              </IconButton>
              <SyntaxHighlighter wrapLines customStyle={{ fontSize: 14 }} language="hcl" style={prismTheme} children={usageInfo} />
            </Box>}
            {tab === 'versions' && <Box marginTop={2}>
              <Suspense fallback={<ListSkeleton rowCount={3} />}>
                <TerraformProviderVersionList fragmentRef={data.provider} />
              </Suspense>
            </Box>}
          </React.Fragment>
        </Box>
      </Box>
    </Box>
  );
}

export default TerraformProviderVersionDetails;
