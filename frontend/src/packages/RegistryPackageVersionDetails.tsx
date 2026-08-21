import { Breadcrumbs, CircularProgress, Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { Suspense } from 'react';
import { PreloadedQuery, usePreloadedQuery } from 'react-relay/hooks';
import TRNButton from '../common/TRNButton';
import Link from '../routes/Link';
import PackageDetailsIndex from './PackageDetailsIndex';
import { RegistryPackageVersionDetailsQuery } from './__generated__/RegistryPackageVersionDetailsQuery.graphql';
import { PageLayoutProvider, usePageLayout } from '@/layout/PageLayoutContext';

const query = graphql`
    query RegistryPackageVersionDetailsQuery($id: String!, $first: Int, $last: Int, $after: String, $before: String) {
      node(id: $id) {
        ... on Package {
          id
          name
          metadata {
              trn
          }
          ...PackageDetailsIndexFragment_package
        }
      }
    }
`;

interface Props {
    queryRef: PreloadedQuery<RegistryPackageVersionDetailsQuery>
}

function RegistryPackageVersionDetails(props: Props) {
    const { queryRef } = props;
    const queryData = usePreloadedQuery<RegistryPackageVersionDetailsQuery>(query, queryRef);
    const pkg = queryData.node;

    return (
        <Box component="main" flexGrow={1} minWidth={0}>
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
                <PageLayoutProvider size="wide">
                    {pkg && <RegistryPackageVersionDetailsIndex pkg={pkg} />}
                    {!pkg && <Box display="flex" justifyContent="center" marginTop={4}>
                        <Typography variant="h6" color="textSecondary">
                            package not found
                        </Typography>
                    </Box>}
                </PageLayoutProvider>
            </Suspense>
        </Box>
    );
}

type PackageNode = NonNullable<RegistryPackageVersionDetailsQuery['response']['node']>;

// The registry page is read-only: versions are created and edited from the owning group, so the only
// header action is the TRN.
function RegistryPackageVersionDetailsIndex({ pkg }: { pkg: PackageNode }) {
    usePageLayout('wide');

    return (
        <PackageDetailsIndex
            fragmentRef={pkg}
            breadcrumbs={
                <Breadcrumbs aria-label="breadcrumb" sx={{ marginBottom: 2 }}>
                    <Link color="inherit" to="/package-registry">package registry</Link>
                    <Typography color="inherit">{pkg.name}</Typography>
                </Breadcrumbs>
            }
            actions={pkg.metadata && <TRNButton trn={pkg.metadata.trn} size="small" />}
        />
    );
}

export default RegistryPackageVersionDetails;
