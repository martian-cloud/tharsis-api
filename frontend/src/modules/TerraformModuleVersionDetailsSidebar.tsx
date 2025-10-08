import { Chip, Link as MuiLink, Tooltip, Typography, TypographyProps, styled, useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import Drawer from '../common/Drawer';
import graphql from 'babel-plugin-relay/macro';
import { useFragment } from 'react-relay/hooks';
import Gravatar from '../common/Gravatar';
import Timestamp from '../common/Timestamp';
import Link from '../routes/Link';
import { TerraformModuleVersionDetailsSidebarFragment_details$key } from './__generated__/TerraformModuleVersionDetailsSidebarFragment_details.graphql';

interface Props {
    fragmentRef: TerraformModuleVersionDetailsSidebarFragment_details$key
    open: boolean
    temporary: boolean
    onClose: () => void
}

export const SidebarWidth = 350;

const Section = styled(Box)(() => ({
    marginBottom: 24,
}));

const FieldLabel = styled(
    ({ ...props }: TypographyProps) => <Typography color="textSecondary" variant="subtitle2" {...props} />
)(() => ({
    fontSize: 16,
    marginBottom: 1,
}));

function TerraformModuleVersionDetailsSidebar(props: Props) {
    const { open, temporary, onClose } = props;
    const theme = useTheme();

    const data = useFragment<TerraformModuleVersionDetailsSidebarFragment_details$key>(
        graphql`
    fragment TerraformModuleVersionDetailsSidebarFragment_details on TerraformModuleVersion
    {
        id
        version
        createdBy
        latest
        shaSum
        metadata {
            createdAt
        }
        module {
            id
            name
            system
            registryNamespace
            private
            repositoryUrl
            groupPath
        }
    }
  `, props.fragmentRef);

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
                <Section>
                    <FieldLabel>Module Version ID</FieldLabel>
                    <Tooltip title={data.id}>
                        <Typography noWrap>
                            {data.id.substring(0, 4)}...{data.id.substring(data.id.length - 4)}
                        </Typography>
                    </Tooltip>
                </Section>
                <Section>
                    <FieldLabel>System</FieldLabel>
                    <Typography>
                        {data.module.system}
                    </Typography>
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
                    <FieldLabel>Digest</FieldLabel>
                    <Tooltip title={data.shaSum}>
                        <Typography noWrap>
                            {data.shaSum.substring(0, 4)}...{data.shaSum.substring(data.shaSum.length - 4)}
                        </Typography>
                    </Tooltip>
                </Section>
                <Section>
                    <FieldLabel>Group</FieldLabel>
                    <Typography component="p" noWrap sx={{ color: theme.palette.primary.main }}>
                        <Link noWrap underline="hover" to={`/groups/${data.module.groupPath}`}>
                            <Typography component="span" noWrap>
                                {data.module.groupPath}
                            </Typography>
                        </Link>
                    </Typography>
                </Section>
                <Section>
                    <FieldLabel>Repository</FieldLabel>
                    {data.module.repositoryUrl && <Typography component="p" noWrap sx={{ color: theme.palette.primary.main }}>
                        <MuiLink noWrap underline="hover" href={data.module.repositoryUrl}>
                            <Typography component="span" noWrap>
                                {data.module.repositoryUrl}
                            </Typography>
                        </MuiLink>
                    </Typography>}
                    {!data.module.repositoryUrl && <Typography>None</Typography>}
                </Section>
            </Box>
        </Drawer>
    );
}

export default TerraformModuleVersionDetailsSidebar;
