import { Box, CircularProgress } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { Suspense } from 'react';
import { useLazyLoadQuery } from 'react-relay/hooks';
import MuiMarkdown from '../../common/Markdown';
import NoResults from '../../common/NoResults';
import { TerraformProviderDocItemCategory, TerraformProviderVersionDocFileQuery } from './__generated__/TerraformProviderVersionDocFileQuery.graphql';

interface Props {
    providerVersionId: string
    category: string
    filename: string
}

const query = graphql`
    query TerraformProviderVersionDocFileQuery($id: String!, $category: TerraformProviderDocItemCategory!, $filename: String!) {
        node(id: $id) {
            ... on TerraformProviderVersion {
                docs {
                    fileData(category: $category, filename: $filename)
                }
            }
        }
    }
`;

function DocFileContent({ providerVersionId, category, filename }: Props) {
    const data = useLazyLoadQuery<TerraformProviderVersionDocFileQuery>(
        query,
        // Category ids are lower_snake_case in the URL; the GraphQL enum is UPPER_SNAKE_CASE.
        { id: providerVersionId, category: category.toUpperCase() as TerraformProviderDocItemCategory, filename },
        { fetchPolicy: 'store-and-network' }
    );

    if (!data.node) {
        return <NoResults>This provider version does not exist</NoResults>;
    }

    if (data.node.docs?.fileData == null) {
        return <NoResults>Documentation file not found</NoResults>;
    }

    return <MuiMarkdown children={data.node.docs.fileData} />;
}

function TerraformProviderVersionDocFile(props: Props) {
    return (
        <Suspense fallback={<Box
            sx={{
                display: "flex",
                justifyContent: "center",
                padding: 2
            }}><CircularProgress size={20} /></Box>}>
            <DocFileContent {...props} />
        </Suspense>
    );
}

export default TerraformProviderVersionDocFile;
