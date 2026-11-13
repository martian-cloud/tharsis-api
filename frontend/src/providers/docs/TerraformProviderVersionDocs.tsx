import { Alert, Box, useTheme } from '@mui/material';
import graphql from 'babel-plugin-relay/macro';
import { useMemo } from 'react';
import { useFragment } from 'react-relay/hooks';
import { useSearchParams } from 'react-router-dom';
import MuiMarkdown from '../../common/Markdown';
import NoResults from '../../common/NoResults';
import { TerraformProviderVersionDocsFragment_details$key } from './__generated__/TerraformProviderVersionDocsFragment_details.graphql';
import TerraformProviderVersionDocFile from './TerraformProviderVersionDocFile';
import TerraformProviderVersionDocsSidebar, { OVERVIEW_ITEM_ID } from './TerraformProviderVersionDocsSidebar';

interface Props {
    fragmentRef: TerraformProviderVersionDocsFragment_details$key
}

// Splits a sidebar id "category/name" into its parts; a bare id with no slash has no name.
function parseSelectedItem(item: string): { category: string, name?: string } {
    const separatorIndex = item.indexOf('/');
    if (separatorIndex === -1) {
        return { category: item };
    }
    return { category: item.substring(0, separatorIndex), name: item.substring(separatorIndex + 1) };
}

// Lower_snake_case ids for every category the GraphQL enum accepts; used to reject unknown URL
// categories before they reach the file query (an invalid enum value trips the error boundary).
const knownCategories = new Set(['overview', 'resources', 'data_sources', 'ephemeral_resources', 'functions', 'guides']);

const categoryLabels: Record<string, string> = {
    resources: 'resources',
    data_sources: 'data sources',
    ephemeral_resources: 'ephemeral resources',
    functions: 'functions',
    guides: 'guides',
};

function TerraformProviderVersionDocs({ fragmentRef }: Props) {
    const theme = useTheme();
    const [searchParams, setSearchParams] = useSearchParams();

    const data = useFragment<TerraformProviderVersionDocsFragment_details$key>(
        graphql`
            fragment TerraformProviderVersionDocsFragment_details on TerraformProviderVersion {
                id
                readme
                readmeUploaded
                docs {
                    processingStatus
                    processingErrorMessage
                    items {
                        filename
                        category
                    }
                }
                ...TerraformProviderVersionDocsSidebarFragment_details
            }
        `,
        fragmentRef
    );

    // URL/sidebar ids are lower_snake_case; map the UPPER_SNAKE_CASE GraphQL enum to match.
    const docItems = useMemo(
        () => (data.docs?.items ?? []).map(item => ({ ...item, category: item.category.toLowerCase() })),
        [data.docs?.items]
    );

    // Navigate to doc files ("category/name"); ignore clicks on category/subcategory folder ids ("cat:"/"group:").
    const onItemChange = (item: string) => {
        if (!item.includes('/') && !knownCategories.has(item)) {
            return;
        }
        searchParams.set('item', item);
        setSearchParams(searchParams, { replace: true });
    };

    const selected = useMemo(() => searchParams.get('item') || OVERVIEW_ITEM_ID, [searchParams]);

    const hasOverviewContent = docItems.length > 0 || !!data.readmeUploaded;

    if (data.docs?.processingStatus === 'PENDING') {
        return (
            <Alert severity="info">
                Documentation has been uploaded and will be available shortly.
            </Alert>
        );
    }

    if (data.docs?.processingStatus === 'FAILED') {
        return (
            <Alert severity="error">
                {data.docs.processingErrorMessage || 'Documentation processing failed for this provider version.'}
            </Alert>
        );
    }

    // Docs are optional, so having neither docs nor a README is not an error.
    if (!hasOverviewContent) {
        return <NoResults>There is no documentation for this provider</NoResults>;
    }

    const { category, name } = parseSelectedItem(selected);

    // Overview with no index doc file falls back to the README, or an empty state when there is none.
    const hasOverviewIndex = docItems.some(item => item.category === 'overview');
    const overviewFallback = category === 'overview' && !hasOverviewIndex
        ? (data.readmeUploaded ? 'readme' : 'empty')
        : null;

    return (
        <Box
            sx={{
                display: "flex",
                width: "100%",

                [theme.breakpoints.down('lg')]: {
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    '& > *': { mb: 4 },
                }
            }}>
            {/* Keyed by version so switching versions resets the tree's expansion state. */}
            <TerraformProviderVersionDocsSidebar key={data.id} fragmentRef={data} onItemChange={onItemChange} />
            <Box
                sx={{
                    flex: 1,
                    pl: 2,
                    borderLeft: `1px solid ${theme.palette.divider}`,
                    minWidth: 0,

                    [theme.breakpoints.down('lg')]: {
                        borderLeft: 'none',
                        paddingLeft: 0,
                        width: '100%'
                    }
                }}>
                {!name ? (
                    docItems.some(item => item.category === category) ? (
                        <NoResults>Select a document from the sidebar</NoResults>
                    ) : (
                        <NoResults>This provider does not have any {categoryLabels[category] ?? category}</NoResults>
                    )
                ) : overviewFallback === 'empty' ? (
                    <NoResults>This provider does not have an overview</NoResults>
                ) : overviewFallback === 'readme' ? (
                    <MuiMarkdown children={data.readme ?? ''} />
                ) : !knownCategories.has(category) ? (
                    <NoResults>No documentation found for {category}</NoResults>
                ) : (
                    <TerraformProviderVersionDocFile providerVersionId={data.id} category={category} filename={name} />
                )}
            </Box>
        </Box>
    );
}

export default TerraformProviderVersionDocs;
