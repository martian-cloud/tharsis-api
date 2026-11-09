package providerregistry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// parsedDocFile is a doc file paired with its content, as nextDocFile returns the two separately.
type parsedDocFile struct {
	docFile providerDocFile
	content string
}

// readAllDocFiles drains the streaming reader into parsed doc files, exercising the real parse path.
func readAllDocFiles(packageReader io.Reader) ([]parsedDocFile, error) {
	reader, err := newBoundedTarReader(packageReader)
	if err != nil {
		return nil, err
	}
	defer reader.close()

	var docFiles []parsedDocFile
	seen := map[string]struct{}{}
	for {
		docFile, content, _, err := nextDocFile(reader, seen)
		if err == io.EOF {
			return docFiles, nil
		}
		if err != nil {
			return nil, err
		}
		docFiles = append(docFiles, parsedDocFile{docFile: docFile, content: content})
	}
}

// hasDocFile reports whether a parsed doc file with the given category and name is present.
func hasDocFile(docFiles []parsedDocFile, category models.TerraformProviderDocItemCategory, name string) bool {
	for _, f := range docFiles {
		if f.docFile.Category == category && f.docFile.Name == name {
			return true
		}
	}
	return false
}

func TestResolveDocFile(t *testing.T) {
	type testCase struct {
		name             string
		entryPath        string
		expectCategory   models.TerraformProviderDocItemCategory
		expectFilename   string
		expectRecognized bool
	}

	testCases := []testCase{
		{
			name:             "root index.md is the overview",
			entryPath:        "index.md",
			expectCategory:   models.TerraformProviderDocItemCategoryOverview,
			expectFilename:   "index.md",
			expectRecognized: true,
		},
		{
			name:             "leading ./ is tolerated",
			entryPath:        "./index.md",
			expectCategory:   models.TerraformProviderDocItemCategoryOverview,
			expectFilename:   "index.md",
			expectRecognized: true,
		},
		{
			name:             "resource doc file",
			entryPath:        "resources/a.md",
			expectCategory:   models.TerraformProviderDocItemCategoryResources,
			expectFilename:   "a.md",
			expectRecognized: true,
		},
		{
			// The on-disk dir stays hyphenated while the category value uses an underscore.
			name:             "data-sources dir maps to the data_sources category",
			entryPath:        "data-sources/b.md",
			expectCategory:   models.TerraformProviderDocItemCategoryDataSources,
			expectFilename:   "b.md",
			expectRecognized: true,
		},
		{
			name:             "guide doc file",
			entryPath:        "guides/start.md",
			expectCategory:   models.TerraformProviderDocItemCategoryGuides,
			expectFilename:   "start.md",
			expectRecognized: true,
		},
		{
			name:             "function doc file",
			entryPath:        "functions/arn_build.md",
			expectCategory:   models.TerraformProviderDocItemCategoryFunctions,
			expectFilename:   "arn_build.md",
			expectRecognized: true,
		},
		{
			name:             "ephemeral-resources dir maps to the ephemeral_resources category",
			entryPath:        "ephemeral-resources/token.md",
			expectCategory:   models.TerraformProviderDocItemCategoryEphemeralResources,
			expectFilename:   "token.md",
			expectRecognized: true,
		},
		{
			// Legacy "<name>.html.md" files keep only "<name>" so the stored name has no ".html".
			name:             "legacy .html.md suffix is reduced to the bare name",
			entryPath:        "guides/getting-started.html.md",
			expectCategory:   models.TerraformProviderDocItemCategoryGuides,
			expectFilename:   "getting-started.md",
			expectRecognized: true,
		},
		{
			// Only the trailing ".md" is stripped; a normal internal dot in the name is preserved.
			name:             "an internal dot in the name is preserved",
			entryPath:        "resources/v1.2.md",
			expectCategory:   models.TerraformProviderDocItemCategoryResources,
			expectFilename:   "v1.2.md",
			expectRecognized: true,
		},
		{name: "a non-index root markdown file is not a doc file", entryPath: "README.md"},
		{
			name:             "backslash separators are normalized to a recognized doc file",
			entryPath:        `resources\a.md`,
			expectCategory:   models.TerraformProviderDocItemCategoryResources,
			expectFilename:   "a.md",
			expectRecognized: true,
		},
		{name: "an underscore data_sources dir does not match the hyphenated category key", entryPath: "data_sources/x.md"},
		{name: "an uppercase .MD extension is not recognized", entryPath: "resources/a.MD"},
		{name: "an absolute path is not a doc file", entryPath: "/index.md"},
		{name: "unsupported category dir is ignored", entryPath: "actions/fn.md"},
		{name: "non-markdown file is ignored", entryPath: "resources/notes.txt"},
		{name: "nested subdirectory is ignored", entryPath: "resources/nested/deep.md"},
		{name: "macOS metadata file is ignored", entryPath: "resources/._a.md"},
		{name: "dot-prefixed directory is ignored", entryPath: ".git/config.md"},
		{name: "a wrapped docs/ root is not the expected layout", entryPath: "docs/index.md"},
		{name: "directory entry itself is not a doc file", entryPath: "resources"},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			category, filename, ok := resolveDocFile(test.entryPath)

			assert.Equal(t, test.expectRecognized, ok)
			if !test.expectRecognized {
				return
			}

			assert.Equal(t, test.expectCategory, category)
			assert.Equal(t, test.expectFilename, filename)
		})
	}
}

func TestIsSparseEntry(t *testing.T) {
	type testCase struct {
		name         string
		header       *tar.Header
		expectSparse bool
	}

	testCases := []testCase{
		{name: "regular file is not sparse", header: &tar.Header{Typeflag: tar.TypeReg}},
		{
			name:   "unrelated PAX records are not sparse",
			header: &tar.Header{Typeflag: tar.TypeReg, PAXRecords: map[string]string{"SCHILY.xattr.user.x": "y"}},
		},
		{name: "old GNU sparse type is sparse", header: &tar.Header{Typeflag: tar.TypeGNUSparse}, expectSparse: true},
		{
			name:         "GNU PAX sparse records are sparse",
			header:       &tar.Header{Typeflag: tar.TypeReg, PAXRecords: map[string]string{"GNU.sparse.major": "1"}},
			expectSparse: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expectSparse, isSparseEntry(test.header))
		})
	}
}

func TestStripFrontMatter(t *testing.T) {
	type testCase struct {
		name                string
		raw                 string
		expectedBody        string
		expectedSubcategory string
		expectWarn          bool
	}

	testCases := []testCase{
		{
			name:                "front-matter is stripped",
			raw:                 "---\nsubcategory: \"Networking\"\n---\n\n# Body\n\ntext",
			expectedBody:        "# Body\n\ntext",
			expectedSubcategory: "Networking",
		},
		{
			name:         "front-matter without recognized fields is stripped",
			raw:          "---\npage_title: \"Some Page\"\n---\n# Body",
			expectedBody: "# Body",
		},
		{
			name:         "no front-matter is returned unchanged",
			raw:          "# Body\n\nno front matter here",
			expectedBody: "# Body\n\nno front matter here",
		},
		{
			name:         "unterminated front-matter is returned unchanged",
			raw:          "---\nsubcategory: \"Networking\"\n# Body",
			expectedBody: "---\nsubcategory: \"Networking\"\n# Body",
		},
		{
			name:                "CRLF front-matter is handled",
			raw:                 "---\r\nsubcategory: \"Storage\"\r\n---\r\n\r\n# Body",
			expectedBody:        "# Body",
			expectedSubcategory: "Storage",
		},
		{
			name:                "lone CR (classic Mac) front-matter is handled",
			raw:                 "---\rsubcategory: \"Storage\"\r---\r\r# Body",
			expectedBody:        "# Body",
			expectedSubcategory: "Storage",
		},
		{
			name:         "empty front-matter block is stripped",
			raw:          "---\n---\n# Body",
			expectedBody: "# Body",
		},
		{
			name:         "empty front-matter block with no body yields empty string",
			raw:          "---\n---\n",
			expectedBody: "",
		},
		{
			name:         "closing delimiter is the line that is exactly --- not one that only starts with it",
			raw:          "---\ntitle: x\n---foo: bar\n---\n# Body",
			expectedBody: "# Body",
		},
		{
			name:                "surrounding whitespace is trimmed from the subcategory",
			raw:                 "---\nsubcategory: \"  Compute  \"\n---\n# Body",
			expectedBody:        "# Body",
			expectedSubcategory: "Compute",
		},
		{
			name:                "overlong subcategory is truncated to the maximum length",
			raw:                 "---\nsubcategory: \"" + strings.Repeat("a", maxProviderDocSubcategoryLength+50) + "\"\n---\n# Body",
			expectedBody:        "# Body",
			expectedSubcategory: strings.Repeat("a", maxProviderDocSubcategoryLength),
		},
		{
			name:                "leading UTF-8 BOM before front-matter is handled",
			raw:                 "\ufeff---\nsubcategory: \"Compute\"\n---\n# Body",
			expectedBody:        "# Body",
			expectedSubcategory: "Compute",
		},
		{
			name:                "leading blank lines before front-matter are handled",
			raw:                 "\n\n---\nsubcategory: \"Compute\"\n---\n\n# Body",
			expectedBody:        "# Body",
			expectedSubcategory: "Compute",
		},
		{
			name:                "closing delimiter with trailing whitespace is recognized",
			raw:                 "---\nsubcategory: \"Compute\"\n--- \t\n# Body",
			expectedBody:        "# Body",
			expectedSubcategory: "Compute",
		},
		{
			name:         "malformed front-matter is stripped with an empty subcategory",
			raw:          "---\nkey: [unterminated\n---\n# Body",
			expectedBody: "# Body",
			expectWarn:   true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			body, subcategory, warn := stripFrontMatter([]byte(test.raw))
			assert.Equal(t, test.expectedBody, body)
			assert.Equal(t, test.expectedSubcategory, subcategory)
			assert.Equal(t, test.expectWarn, warn != nil)
		})
	}
}

type docsEntry struct {
	path       string
	linkTarget string
	contents   []byte
	paxGlobal  bool
	sparse     bool
}

// makeDocsPackage builds an in-memory gzip+tar archive matching the format the docs package reader reads.
func makeDocsPackage(t *testing.T, entries []docsEntry) io.Reader {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, e := range entries {
		switch {
		case e.sparse:
			require.NoError(t, tw.WriteHeader(&tar.Header{
				Name:     e.path,
				Typeflag: tar.TypeGNUSparse,
				Mode:     0600,
				Format:   tar.FormatGNU,
			}))
		case e.paxGlobal:
			require.NoError(t, tw.WriteHeader(&tar.Header{
				Name:       e.path,
				Typeflag:   tar.TypeXGlobalHeader,
				Format:     tar.FormatPAX,
				PAXRecords: map[string]string{"comment": "test"},
			}))
		case e.linkTarget != "":
			require.NoError(t, tw.WriteHeader(&tar.Header{
				Name:     e.path,
				Typeflag: tar.TypeSymlink,
				Linkname: e.linkTarget,
				Mode:     0777,
			}))
		case len(e.path) > 0 && e.path[len(e.path)-1] == '/':
			require.NoError(t, tw.WriteHeader(&tar.Header{Name: e.path, Typeflag: tar.TypeDir, Mode: 0700}))
		default:
			require.NoError(t, tw.WriteHeader(&tar.Header{
				Name:     e.path,
				Typeflag: tar.TypeReg,
				Mode:     0600,
				Size:     int64(len(e.contents)),
			}))
			_, err := tw.Write(e.contents)
			require.NoError(t, err)
		}
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())

	return &buf
}

func TestReadDocsPackage(t *testing.T) {
	// manyFiles builds one more regular-file entry than the entry-count cap allows.
	manyFiles := make([]docsEntry, 0, maxProviderDocsFileCount+1)
	for i := 0; i < maxProviderDocsFileCount+1; i++ {
		manyFiles = append(manyFiles, docsEntry{path: "resources/r" + strconv.Itoa(i) + ".md", contents: []byte("x")})
	}

	// manyNonFiles builds one more non-regular (symlink) entry than the cap allows; it only trips if
	// skipped entries are counted too, which a doc-file-only count would miss.
	manyNonFiles := make([]docsEntry, 0, maxProviderDocsFileCount+1)
	for i := 0; i < maxProviderDocsFileCount+1; i++ {
		manyNonFiles = append(manyNonFiles, docsEntry{path: "resources/link" + strconv.Itoa(i), linkTarget: "a.md"})
	}

	type testCase struct {
		name           string
		entries        []docsEntry
		expectDocFiles map[models.TerraformProviderDocItemCategory][]string
		expectErrCode  errors.CodeType
	}

	testCases := []testCase{
		{
			name: "reads every recognized doc file, ignoring directories and unsupported entries",
			entries: []docsEntry{
				{path: "index.md", contents: []byte("# overview")},
				{path: "resources/"},
				{path: "resources/a.md", contents: []byte("# resource a")},
				{path: "data-sources/b.md", contents: []byte("# data source b")},
				{path: "ephemeral-resources/e.md", contents: []byte("# ephemeral resource e")},
				{path: "functions/fn.md", contents: []byte("# function fn")},
				{path: "guides/start.md", contents: []byte("# guide")},
				// Ignored: unsupported category, non-markdown, nested, and macOS metadata files.
				{path: "actions/act.md", contents: []byte("# out of scope")},
				{path: "resources/notes.txt", contents: []byte("ignored")},
				{path: "resources/nested/deep.md", contents: []byte("# too deep")},
				{path: "resources/._a.md", contents: []byte("junk")},
			},
			expectDocFiles: map[models.TerraformProviderDocItemCategory][]string{
				models.TerraformProviderDocItemCategoryOverview:           {"index.md"},
				models.TerraformProviderDocItemCategoryResources:          {"a.md"},
				models.TerraformProviderDocItemCategoryDataSources:        {"b.md"},
				models.TerraformProviderDocItemCategoryEphemeralResources: {"e.md"},
				models.TerraformProviderDocItemCategoryFunctions:          {"fn.md"},
				models.TerraformProviderDocItemCategoryGuides:             {"start.md"},
			},
		},
		{
			name: "only root index.md becomes the overview; other root markdown files are ignored",
			entries: []docsEntry{
				{path: "index.md", contents: []byte("# overview")},
				{path: "guide.md", contents: []byte("# not an overview doc file")},
				{path: "README.md", contents: []byte("# readme, not a doc file")},
				{path: "resources/a.md", contents: []byte("# resource a")},
			},
			expectDocFiles: map[models.TerraformProviderDocItemCategory][]string{
				models.TerraformProviderDocItemCategoryOverview:  {"index.md"},
				models.TerraformProviderDocItemCategoryResources: {"a.md"},
			},
		},
		{
			// A symlink entry is skipped rather than rejected, so an archive produced by the CLI's
			// packer (which emits in-root symlinks as-is) still processes its real doc files.
			name: "skips a symlink entry instead of failing the archive",
			entries: []docsEntry{
				{path: "index.md", contents: []byte("# overview")},
				{path: "resources/alias.md", linkTarget: "a.md"},
				{path: "resources/a.md", contents: []byte("# resource a")},
			},
			expectDocFiles: map[models.TerraformProviderDocItemCategory][]string{
				models.TerraformProviderDocItemCategoryOverview:  {"index.md"},
				models.TerraformProviderDocItemCategoryResources: {"a.md"},
			},
		},
		{
			// A pax global header (emitted by git archive) must not fail the upload.
			name: "skips a pax global header",
			entries: []docsEntry{
				{path: "pax_global_header", paxGlobal: true},
				{path: "index.md", contents: []byte("# overview")},
			},
			expectDocFiles: map[models.TerraformProviderDocItemCategory][]string{
				models.TerraformProviderDocItemCategoryOverview: {"index.md"},
			},
		},
		{
			name:           "an archive with no recognized doc files yields no doc files and no error",
			entries:        []docsEntry{{path: "actions/fn.md", contents: []byte("# out of scope")}},
			expectDocFiles: map[models.TerraformProviderDocItemCategory][]string{},
		},
		{
			name:          "rejects a single doc file over the per-file cap",
			entries:       []docsEntry{{path: "resources/big.md", contents: make([]byte, maxProviderDocFileSize+1)}},
			expectErrCode: errors.ETooLarge,
		},
		{
			name:          "rejects a decompression bomb hidden in a skipped entry (size cap)",
			entries:       []docsEntry{{path: "actions/fn.md", contents: make([]byte, maxProviderDocsUncompressedSize+1)}},
			expectErrCode: errors.ETooLarge,
		},
		{
			name:          "rejects too many entries (entry-count cap)",
			entries:       manyFiles,
			expectErrCode: errors.ETooLarge,
		},
		{
			name:          "rejects too many non-regular entries",
			entries:       manyNonFiles,
			expectErrCode: errors.ETooLarge,
		},
		{
			name:          "rejects a path-traversal entry",
			entries:       []docsEntry{{path: "../escape.md", contents: []byte("nope")}},
			expectErrCode: errors.EInvalid,
		},
		{
			name:          "rejects a sparse entry",
			entries:       []docsEntry{{path: "resources/sparse.md", sparse: true}},
			expectErrCode: errors.EInvalid,
		},
		{
			name: "rejects an archive with more than one entry for the same doc file",
			entries: []docsEntry{
				{path: "resources/a.md", contents: []byte("# first")},
				{path: "./resources/a.md", contents: []byte("# second")},
			},
			expectErrCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			docFiles, err := readAllDocFiles(makeDocsPackage(t, test.entries))

			if test.expectErrCode != "" {
				require.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			expectedTotal := 0
			for category, names := range test.expectDocFiles {
				expectedTotal += len(names)
				for _, name := range names {
					assert.True(t, hasDocFile(docFiles, category, name), "expected %s doc file %q", category, name)
				}
			}
			assert.Len(t, docFiles, expectedTotal)
		})
	}
}

// A non-gzip stream must surface as EInvalid, not the EInternal that uncoded errors default to.
func TestReadDocsPackageInvalidGzip(t *testing.T) {
	_, err := readAllDocFiles(bytes.NewReader([]byte("this is not a gzip stream")))
	require.Equal(t, errors.EInvalid, errors.ErrorCode(err))
}

// Data after tar's end-of-archive marker is still decompressed by the drain, so it must count toward the cap.
func TestReadDocsPackageTrailingDataExceedsCap(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "index.md", Mode: 0600, Size: 4, Typeflag: tar.TypeReg}))
	_, err := tw.Write([]byte("# hi"))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	_, err = gz.Write(make([]byte, maxProviderDocsUncompressedSize+1))
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	_, err = readAllDocFiles(&buf)
	require.Equal(t, errors.ETooLarge, errors.ErrorCode(err))
}

// TestReadDocsPackageStripsFrontMatter verifies doc file content reaches the caller with YAML
// front-matter removed, since that stripping now happens inline while reading the archive.
func TestReadDocsPackageStripsFrontMatter(t *testing.T) {
	docFiles, err := readAllDocFiles(makeDocsPackage(t, []docsEntry{
		{path: "resources/a.md", contents: []byte("---\nsubcategory: \"Networking\"\n---\n\n# resource a")},
	}))
	require.NoError(t, err)
	require.Len(t, docFiles, 1)

	assert.Equal(t, "a.md", docFiles[0].docFile.Name)
	assert.Equal(t, "Networking", docFiles[0].docFile.Subcategory)
	assert.Equal(t, "# resource a", docFiles[0].content)
}
