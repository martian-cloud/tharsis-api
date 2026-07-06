import { Box, Paper, Typography, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useSnackbar } from 'notistack';
import { useState } from 'react';
import { useFragment, useMutation } from "react-relay/hooks";
import ConfirmationDialog from '../../common/ConfirmationDialog';
import { ResponsiveTable } from '../../common/ResponsiveTable';
import NamespaceMembershipListItem from './NamespaceMembershipListItem';
import { NamespaceMembershipListDeleteNamespaceMembershipMutation } from './__generated__/NamespaceMembershipListDeleteNamespaceMembershipMutation.graphql';
import { NamespaceMembershipListFragment_memberships$key } from './__generated__/NamespaceMembershipListFragment_memberships.graphql';

const getMemberName = (membership: any) => {
  switch (membership.member.__typename) {
    case 'User':
      return membership.member.username;
    case 'Team':
      return membership.member.name;
    case 'ServiceAccount':
      return membership.member.resourcePath;
    default:
      return '';
  }
};

const membershipSearchFilter = (search: string) => (membership: any) => {
  switch (membership.member.__typename) {
    case 'User':
      return membership.member.username.toLowerCase().startsWith(search);
    case 'Team':
      return membership.member.name.toLowerCase().startsWith(search);
    case 'ServiceAccount':
      return membership.member.name.toLowerCase().startsWith(search) ||
        membership.member.resourcePath.toLowerCase().startsWith(search);
    default:
      return '';
  }
}

interface Props {
  fragmentRef: NamespaceMembershipListFragment_memberships$key
  search: string
}

function NamespaceMembershipList(props: Props) {
  const { fragmentRef, search } = props;

  const theme = useTheme();
  const { enqueueSnackbar } = useSnackbar();

  const data = useFragment<NamespaceMembershipListFragment_memberships$key>(
    graphql`
    fragment NamespaceMembershipListFragment_memberships on Namespace
    {
      fullPath
      memberships {
          id
          member {
              __typename
              ...on User {
                  username
                  email
              }
              ...on Team {
                name
            }
              ...on ServiceAccount {
                  resourcePath
                  name
              }
          }
          ...NamespaceMembershipListItemFragment_membership
      }
    }
  `, fragmentRef);

  const [commitDeleteNamespaceMembership, deleteInFlight] = useMutation<NamespaceMembershipListDeleteNamespaceMembershipMutation>(graphql`
        mutation NamespaceMembershipListDeleteNamespaceMembershipMutation($input: DeleteNamespaceMembershipInput!) {
            deleteNamespaceMembership(input: $input) {
                namespace {
                    memberships {
                        ...NamespaceMembershipListItemFragment_membership
                    }
                }
                problems {
                    message
                    field
                    type
                }
            }
        }
    `);

  const [membershipToDelete, setNamespaceMembershipToDelete] = useState<any>(null);

  const onShowDeleteConfirmationDialog = (membership: any) => {
    setNamespaceMembershipToDelete(membership);
  };

  const onDelete = (confirm?: boolean) => {
    if (confirm && membershipToDelete) {
      commitDeleteNamespaceMembership({
        variables: {
          input: {
            id: membershipToDelete.id
          },
        },
        onCompleted: data => {
          if (data.deleteNamespaceMembership.problems.length) {
            enqueueSnackbar(data.deleteNamespaceMembership.problems.map(problem => problem.message).join('; '), { variant: 'warning' });
          }
          setNamespaceMembershipToDelete(null);
        },
        onError: error => {
          enqueueSnackbar(`Unexpected error occurred: ${error.message}`, { variant: 'error' });
          setNamespaceMembershipToDelete(null);
        }
      });
    } else {
      setNamespaceMembershipToDelete(null);
    }
  };

  const filteredNamespaceMemberships = search ? data.memberships.filter(membershipSearchFilter(search)) : [...data.memberships];
  filteredNamespaceMemberships.sort((a: any, b: any) => {
    const n1 = getMemberName(a);
    const n2 = getMemberName(b);
    return n1.localeCompare(n2);
  });

  return (
    <Box>
      {(search !== '' || filteredNamespaceMemberships.length > 0) && <Box>
        <Paper sx={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0, border: `1px solid ${theme.palette.divider}` }}>
          <Box padding={2} display="flex" alignItems="center" justifyContent="space-between">
            <Typography variant="subtitle1">{filteredNamespaceMemberships.length} member{filteredNamespaceMemberships.length === 1 ? '' : 's'}</Typography>
          </Box>
        </Paper>
        {filteredNamespaceMemberships.length > 0 && <ResponsiveTable
          ariaLabel="memberships"
          minWidth={650}
          columns={[
            { label: 'Name' },
            { label: 'Type' },
            { label: 'Role' },
            { label: 'Last Updated' },
            { label: 'Source' },
            { label: '' },
          ]}
        >
          {filteredNamespaceMemberships.map((membership: any) => <NamespaceMembershipListItem
            key={membership.id}
            fragmentRef={membership}
            namespacePath={data.fullPath}
            onDelete={onShowDeleteConfirmationDialog}
          />)}
        </ResponsiveTable>}
      </Box>}
      {filteredNamespaceMemberships.length === 0 && <Typography color="textSecondary" align="center" sx={{
        padding: 4,
        borderBottom: `1px solid ${theme.palette.divider}`,
        borderLeft: `1px solid ${theme.palette.divider}`,
        borderRight: `1px solid ${theme.palette.divider}`,
        borderBottomLeftRadius: 4,
        borderBottomRightRadius: 4
      }}>
        No members matching search <strong>{search}</strong>
      </Typography>}
      {membershipToDelete && (
        <ConfirmationDialog
          title="Remove Member"
          confirmLabel="Remove"
          confirmInProgress={deleteInFlight}
          onConfirm={() => onDelete(true)}
          onClose={() => onDelete()}
        >
          Are you sure you want to remove the member <strong>{getMemberName(membershipToDelete)}</strong>?
        </ConfirmationDialog>
      )}
    </Box>
  );
}

export default NamespaceMembershipList;
