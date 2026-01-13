import { useState, useEffect } from "react";
import { Box, Tooltip } from "@mui/material";
import { FavoriteBorder } from "@mui/icons-material";
import { LoadingButton } from "@mui/lab";
import { useMutation, useLazyLoadQuery } from "react-relay/hooks";
import { useSnackbar } from "notistack";
import graphql from "babel-plugin-relay/macro";
import { NamespaceFavoriteButtonCreateMutation } from "./__generated__/NamespaceFavoriteButtonCreateMutation.graphql";
import { NamespaceFavoriteButtonDeleteMutation } from "./__generated__/NamespaceFavoriteButtonDeleteMutation.graphql";
import { NamespaceFavoriteButtonQuery } from "./__generated__/NamespaceFavoriteButtonQuery.graphql";

interface Props {
    namespacePath: string;
    namespaceType: "GROUP" | "WORKSPACE";
}

const query = graphql`
    query NamespaceFavoriteButtonQuery($namespacePath: String!) {
        me {
            ... on User {
                namespaceFavorites(first: 1, namespacePath: $namespacePath) {
                    edges {
                        node {
                            id
                        }
                    }
                }
            }
        }
    }
`;

function NamespaceFavoriteButton({ namespacePath, namespaceType }: Props) {
    const data = useLazyLoadQuery<NamespaceFavoriteButtonQuery>(query, { namespacePath }, { fetchPolicy: 'store-and-network' });
    
    const [isNamespaceFavorite, setIsNamespaceFavorite] = useState<boolean>(false);
    const { enqueueSnackbar } = useSnackbar();

    useEffect(() => {
        setIsNamespaceFavorite((data?.me?.namespaceFavorites?.edges?.length ?? 0) > 0);
    }, [data]);

    const [commitFavorite, favoriteInFlight] = useMutation<NamespaceFavoriteButtonCreateMutation>(graphql`
        mutation NamespaceFavoriteButtonCreateMutation($input: NamespaceFavoriteInput!) {
            favoriteNamespace(input: $input) {
                namespaceFavorite {
                    id
                }
                problems {
                    message
                    type
                }
            }
        }
    `);

    const [commitUnfavorite, unfavoriteInFlight] = useMutation<NamespaceFavoriteButtonDeleteMutation>(graphql`
        mutation NamespaceFavoriteButtonDeleteMutation($input: NamespaceFavoriteInput!) {
            unfavoriteNamespace(input: $input) {
                problems {
                    message
                    type
                }
            }
        }
    `);

    const handleToggle = () => {
        if (!isNamespaceFavorite) {
            commitFavorite({
                variables: {
                    input: {
                        namespacePath,
                        namespaceType,
                    },
                },
                onCompleted: () => {
                    setIsNamespaceFavorite(true);
                },
                onError: () => {
                    enqueueSnackbar('Failed to add to favorites', { variant: 'error' });
                },
            });
        } else {
            commitUnfavorite({
                variables: {
                    input: {
                        namespacePath,
                        namespaceType,
                    },
                },
                onCompleted: () => {
                    setIsNamespaceFavorite(false);
                },
                onError: () => {
                    enqueueSnackbar('Failed to remove from favorites', { variant: 'error' });
                },
            });
        }
    };

    const loading = favoriteInFlight || unfavoriteInFlight;

    return (
        <Box>
            <Tooltip title={isNamespaceFavorite ? "Remove from favorites" : "Add to favorites"}>
                <LoadingButton
                    variant="outlined"
                    size="small"
                    color={isNamespaceFavorite ? "primary" : "info"}
                    onClick={handleToggle}
                    loading={loading}
                    sx={{
                        minWidth: "auto",
                        px: 1
                    }}
                >
                    <FavoriteBorder fontSize="medium" />
                </LoadingButton>
            </Tooltip>
        </Box>
    );
}

export default NamespaceFavoriteButton;
