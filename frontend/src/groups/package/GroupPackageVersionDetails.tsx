import { usePageLayout } from '@/layout/PageLayoutContext';
import { Typography } from '@mui/material';
import Box from '@mui/material/Box';
import graphql from 'babel-plugin-relay/macro';
import { useLazyLoadQuery } from 'react-relay/hooks';
import { useParams } from 'react-router-dom';
import TRNButton from '../../common/TRNButton';
import NamespaceBreadcrumbs from '../../namespace/NamespaceBreadcrumbs';
import PackageDetailsIndex from '../../packages/PackageDetailsIndex';
import { GroupPackageVersionDetailsQuery } from './__generated__/GroupPackageVersionDetailsQuery.graphql';

const INITIAL_ITEM_COUNT = 20;

const query = graphql`
    query GroupPackageVersionDetailsQuery($id: String!, $first: Int, $last: Int, $after: String, $before: String) {
      node(id: $id) {
        ... on Package {
          id
          name
          groupPath
          metadata {
              trn
          }
          ...PackageDetailsIndexFragment_package
        }
      }
    }
`;

function GroupPackageVersionDetails() {
    const { packageId } = useParams();

    const queryData = useLazyLoadQuery<GroupPackageVersionDetailsQuery>(
        query,
        { id: packageId as string, first: INITIAL_ITEM_COUNT },
        { fetchPolicy: 'store-and-network' }
    );
    const pkg = queryData.node;

    return (
        <Box component="main" flexGrow={1} minWidth={0}>
            {pkg && <GroupPackageVersionDetailsIndex pkg={pkg} />}
            {!pkg && <Box display="flex" justifyContent="center" marginTop={4}>
                <Typography variant="h6" color="textSecondary">
                    package not found
                </Typography>
            </Box>}
        </Box>
    );
}

type PackageNode = NonNullable<GroupPackageVersionDetailsQuery['response']['node']>;

// The group owns the package, so this is where versions are created and edited.
function GroupPackageVersionDetailsIndex({ pkg }: { pkg: PackageNode }) {
    usePageLayout('wide');

    return (
        <PackageDetailsIndex
            fragmentRef={pkg}
            breadcrumbs={
                <NamespaceBreadcrumbs
                    namespacePath={pkg.groupPath ?? ''}
                    childRoutes={[
                        { title: 'packages', path: 'packages' },
                        { title: pkg.name ?? '', path: `${pkg.id}`, disabled: true }
                    ]}
                />
            }
            // The create / edit / delete control acts on the version being viewed, so
            // PackageDetailsIndex renders it; only the TRN is a plain header action.
            canManageVersions
            actions={pkg.metadata && <TRNButton trn={pkg.metadata.trn} size="small" />}
        />
    );
}

export default GroupPackageVersionDetails;
