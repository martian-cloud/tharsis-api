import { Chip, Link, Tooltip, Typography, TypographyProps, styled, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import Drawer from '../common/Drawer';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import { TerraformProviderVersionDetailsSidebarFragment_details$key } from './__generated__/TerraformProviderVersionDetailsSidebarFragment_details.graphql';

interface Props {
  fragmentRef: TerraformProviderVersionDetailsSidebarFragment_details$key
  open: boolean
  temporary: boolean
  onClose: () => void
}

export const SidebarWidth = 400;

const Section = styled(Box)(() => ({
  marginBottom: 24,
}));

const FieldLabel = styled(
  ({ ...props }: TypographyProps) => <Typography color="textSecondary" variant="subtitle2" {...props} />
)(() => ({
  fontSize: 16,
  marginBottom: 1,
}));

function TerraformProviderVersionDetailsSidebar(props: Props) {
  const { open, temporary, onClose } = props;
  const theme = useTheme();

  const data = useFragment<TerraformProviderVersionDetailsSidebarFragment_details$key>(
    graphql`
    fragment TerraformProviderVersionDetailsSidebarFragment_details on TerraformProviderVersion
    {
      version
      createdBy
      gpgKeyId
      protocols
      latest
      platforms {
        id
        os
        arch
        binaryUploaded
      }
      metadata {
          createdAt
      }
      provider {
        id
        name
        registryNamespace
        private
        repositoryUrl
      }
    }
  `, props.fragmentRef);

  const filteredPlatforms = data.platforms.filter(platform => platform.binaryUploaded).map(platform => `${platform.os}_${platform.arch}`);
  filteredPlatforms.sort();

  return (
    <Drawer
      width={SidebarWidth}
      temporary={temporary}
      variant={temporary ? 'temporary' : 'permanent'}
      open={open}
      hideBackdrop={false}
      anchor='right'
      onClose={onClose}
    >
      <Box padding={2}>
        <Section>
          <FieldLabel>Version</FieldLabel>
          <Box display="flex" alignItems="center">
            <Typography>
              {data.version}
            </Typography>
            {data.latest && <Chip size="small" color="secondary" sx={{ marginLeft: 1 }} label="latest" />}
          </Box>
        </Section>
        {data && <Section>
          <FieldLabel>Published</FieldLabel>
          <Box display="flex" alignItems="center">
            <Typography sx={{ marginRight: 1 }}>
              <Timestamp component="span" timestamp={data.metadata.createdAt} /> by
            </Typography>
            <Tooltip title={data.createdBy}>
              <Box>
                <Gravatar width={20} height={20} email={data.createdBy} />
              </Box>
            </Tooltip>
          </Box>
        </Section>}
        <Section>
          <FieldLabel>Repository</FieldLabel>
          {data.provider.repositoryUrl && <Typography component="p" noWrap sx={{ color: theme.palette.primary.main }}>
            <Link noWrap underline="hover" href={data.provider.repositoryUrl}>
              <Typography component="span" noWrap>
                {data.provider.repositoryUrl}
              </Typography>
            </Link>
          </Typography>}
          {!data.provider.repositoryUrl && <Typography>None</Typography>}
        </Section>
        <Section>
          <FieldLabel>GPG Key ID</FieldLabel>
          <Typography>
            {data.gpgKeyId ? data.gpgKeyId : 'None'}
          </Typography>
        </Section>
        <Section>
          <FieldLabel>Protocols</FieldLabel>
          {data.protocols.length > 0 && <Box
            display="flex"
            flexWrap="wrap"
            sx={{
              margin: '0 -4px',
              '& > *': {
                margin: '4px'
              },
            }}
          >
            {data.protocols.map((protocol: string) => (
              <Chip
                size="small"
                key={protocol}
                variant="outlined"
                label={protocol}
              />
            ))}
          </Box>}
        </Section>
        <Section>
          <FieldLabel>Platforms</FieldLabel>
          {filteredPlatforms.length > 0 && <Box
            display="flex"
            flexWrap="wrap"
            sx={{
              margin: '0 -4px',
              '& > *': {
                margin: '4px'
              },
            }}
          >
            {filteredPlatforms.map((platform: any) => (
              <Chip
                size="small"
                key={platform}
                variant="outlined"
                label={platform}
              />
            ))}
          </Box>}
        </Section>
      </Box>
    </Drawer>
  );
}

export default TerraformProviderVersionDetailsSidebar;
