// Package providerregistry contains the service-agnostic Terraform provider registry docs logic for
// parsing and processing. It must not import any internal/services/* package.
package providerregistry

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"path"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/units"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/sanitize"
	"gopkg.in/yaml.v3"
)

// categoryDirs maps each docs directory to its category; keys stay hyphenated as they appear on disk even
// though the category value uses an underscore. The root (empty dir) is the overview; unlisted dirs are ignored.
var categoryDirs = map[string]models.TerraformProviderDocItemCategory{
	"":                    models.TerraformProviderDocItemCategoryOverview,
	"resources":           models.TerraformProviderDocItemCategoryResources,
	"data-sources":        models.TerraformProviderDocItemCategoryDataSources,
	"ephemeral-resources": models.TerraformProviderDocItemCategoryEphemeralResources,
	"functions":           models.TerraformProviderDocItemCategoryFunctions,
	"guides":              models.TerraformProviderDocItemCategoryGuides,
}

const (
	// maxProviderDocFileSize caps a single doc file; content is held in memory, so this also bounds per-file memory.
	maxProviderDocFileSize = 500 * units.KiB

	// maxProviderDocsUncompressedSize bounds total decompressed bytes read from a docs package.
	maxProviderDocsUncompressedSize = 64 * units.MiB

	// maxProviderDocsFileCount bounds the number of archive entries read from a docs package.
	maxProviderDocsFileCount = 10000

	// maxProviderDocSubcategoryLength caps a doc file's subcategory; the value is a display-only grouping label
	maxProviderDocSubcategoryLength = 255
)

// errOverMaxUncompressedSize is the shared over-size error, raised once the gzip byte cap is exhausted.
var errOverMaxUncompressedSize = errors.New(
	"docs package exceeds the maximum uncompressed size of %d bytes", maxProviderDocsUncompressedSize,
	errors.WithErrorCode(errors.ETooLarge),
)

// providerDocFile is a single parsed documentation file's metadata; its content is passed separately so a
// whole package is never held in memory at once.
type providerDocFile struct {
	Category    models.TerraformProviderDocItemCategory
	Name        string
	Subcategory string
}

// nextDocFile returns the next recognized markdown entry's metadata and content, a non-fatal warn (e.g. a
// malformed front-matter block), or io.EOF when exhausted; seen rejects duplicate category/name keys.
func nextDocFile(reader *boundedTarReader, seen map[string]struct{}) (providerDocFile, string, error, error) {
	for {
		name, err := reader.next()
		if err != nil {
			return providerDocFile{}, "", nil, err
		}

		category, filename, isDocFile := resolveDocFile(name)
		if !isDocFile {
			continue
		}

		docFileKey := string(category) + "/" + filename
		if _, ok := seen[docFileKey]; ok {
			return providerDocFile{}, "", nil, errors.New(
				"docs package contains more than one entry for documentation file %s", docFileKey,
				errors.WithErrorCode(errors.EInvalid),
			)
		}
		seen[docFileKey] = struct{}{}

		fileContent, err := reader.readEntry(filename)
		if err != nil {
			return providerDocFile{}, "", nil, err
		}

		body, subcategory, warn := stripFrontMatter(fileContent)

		return providerDocFile{Category: category, Name: filename, Subcategory: subcategory}, body, warn, nil
	}
}

// boundedTarReader reads a gzip+tar stream bounded by a gzip-level byte cap, a per-entry cap, an entry-count cap, and a relative-path check.
type boundedTarReader struct {
	gz         *gzip.Reader
	limited    *io.LimitedReader
	tr         *tar.Reader
	entryCount int
}

// newBoundedTarReader wraps the reader in a gzip+tar pipeline bounded by the package-level caps.
func newBoundedTarReader(r io.Reader) (*boundedTarReader, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open docs package gzip stream", errors.WithErrorCode(errors.EInvalid))
	}

	// Bound decompressed bytes at the gzip layer so even skipped entries can't inflate unbounded; +1 detects overflow.
	limited := &io.LimitedReader{R: gz, N: maxProviderDocsUncompressedSize + 1}

	return &boundedTarReader{
		gz:      gz,
		limited: limited,
		tr:      tar.NewReader(limited),
	}, nil
}

// next returns the next regular-file entry's name, skipping (but still counting and path-checking) other entries.
func (b *boundedTarReader) next() (name string, err error) {
	for {
		hdr, err := b.tr.Next()
		if err == io.EOF {
			// Drain the rest so gzip verifies its trailer; tar stops at its own end-of-archive marker.
			if _, drainErr := io.Copy(io.Discard, b.limited); drainErr != nil {
				return "", b.classifyReadError(drainErr)
			}
			// Checked after the drain so data past the end-of-archive marker still counts toward the cap.
			if b.limited.N <= 0 {
				return "", errOverMaxUncompressedSize
			}
			return "", io.EOF
		}
		if err != nil {
			return "", b.classifyReadError(err)
		}

		// Count every entry type, before the regular-file check, so skipped entries still count.
		b.entryCount++
		if b.entryCount > maxProviderDocsFileCount {
			return "", errors.New(
				"docs package exceeds the maximum entry count of %d", maxProviderDocsFileCount,
				errors.WithErrorCode(errors.ETooLarge),
			)
		}

		if !isSafeEntryName(hdr.Name) {
			return "", errors.New(
				"docs package entry %q is not a valid relative path", hdr.Name,
				errors.WithErrorCode(errors.EInvalid),
			)
		}

		// archive/tar fills sparse holes with zeros after the gzip-level cap, so a sparse entry could inflate past it.
		if isSparseEntry(hdr) {
			return "", errors.New(
				"docs package entry %q is a sparse file, which is not supported", hdr.Name,
				errors.WithErrorCode(errors.EInvalid),
			)
		}

		// Only regular files can be doc files; skip everything else and advance to the next entry.
		if hdr.Typeflag == tar.TypeReg {
			return hdr.Name, nil
		}
	}
}

// readEntry reads the current entry's content into memory, rejecting it if it exceeds the per-file cap.
func (b *boundedTarReader) readEntry(filename string) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(b.tr, maxProviderDocFileSize+1))
	if err != nil {
		return nil, b.classifyReadError(errors.Wrap(err, "failed to read doc file %s", filename))
	}

	if int64(len(content)) > maxProviderDocFileSize {
		return nil, errors.New(
			"documentation file %q exceeds the maximum size of %d bytes", filename, maxProviderDocFileSize,
			errors.WithErrorCode(errors.ETooLarge),
		)
	}

	return content, nil
}

func (b *boundedTarReader) close() {
	_ = b.gz.Close()
}

// classifyReadError returns the over-size error when the decompression cap was exhausted, otherwise an
// EInvalid error wrapping the read failure.
func (b *boundedTarReader) classifyReadError(err error) error {
	if b.limited.N <= 0 {
		return errOverMaxUncompressedSize
	}
	return errors.Wrap(err, "failed to read docs package tar stream", errors.WithErrorCode(errors.EInvalid))
}

// normalizeEntrySeparators converts tar entry backslashes to forward slashes for consistent path handling.
func normalizeEntrySeparators(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}

// isSafeEntryName rejects absolute and parent-escaping tar entry names so a crafted archive can't
// smuggle an unexpected path into doc file classification (and from there into an object storage key).
func isSafeEntryName(name string) bool {
	normalized := normalizeEntrySeparators(name)
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return false
	}

	for _, part := range strings.Split(path.Clean(normalized), "/") {
		if part == ".." {
			return false
		}
	}

	return true
}

// isSparseEntry reports whether a tar entry is a sparse file, in the old GNU or the GNU PAX format.
func isSparseEntry(hdr *tar.Header) bool {
	if hdr.Typeflag == tar.TypeGNUSparse {
		return true
	}

	for key := range hdr.PAXRecords {
		if strings.HasPrefix(key, "GNU.sparse.") {
			return true
		}
	}

	return false
}

// resolveDocFile maps an archive entry to its (category, file-name) under the index.md/resources/data-sources/guides layout, returning false otherwise.
func resolveDocFile(entryPath string) (models.TerraformProviderDocItemCategory, string, bool) {
	cleaned := path.Clean(normalizeEntrySeparators(entryPath))

	for _, part := range strings.Split(cleaned, "/") {
		// A dotfile anywhere in the path (including ".." components) disqualifies the entry.
		if part == "" || strings.HasPrefix(part, ".") {
			return "", "", false
		}
	}

	dir, file := path.Split(cleaned)
	category, ok := categoryDirs[strings.TrimSuffix(dir, "/")]
	if !ok || path.Ext(file) != ".md" {
		return "", "", false
	}

	// The overview is a single fixed doc file: only the root index.md maps to it.
	if category == models.TerraformProviderDocItemCategoryOverview && file != "index.md" {
		return "", "", false
	}

	// The stored name keeps its extension so future formats stay distinguishable; a legacy
	// "<name>.html.md" is normalized to "<name>.md".
	name := file
	if base, ok := strings.CutSuffix(file, ".html.md"); ok {
		name = base + ".md"
	}

	return category, name, true
}

// stripFrontMatter drops the optional "---" front-matter block; malformed YAML is still stripped and returned as warn so one bad page can't fail the package.
func stripFrontMatter(raw []byte) (body string, subcategory string, warn error) {
	const delimiter = "---"

	// Strip a leading UTF-8 BOM so the front-matter prefix check still matches.
	normalized := strings.TrimPrefix(string(raw), "\ufeff")

	// Normalize CRLF and lone CR to LF so the "---" line match is platform-independent.
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	// Front-matter may be preceded by blank lines; the opening "---" only needs to be the first non-blank line.
	trimmed := strings.TrimLeft(normalized, "\n")
	if !strings.HasPrefix(trimmed, delimiter+"\n") {
		return normalized, "", nil
	}

	// Scan for the closing "---" line, accepting trailing whitespace, without splitting the whole body.
	rest := trimmed[len(delimiter)+1:]
	for offset := 0; offset <= len(rest); {
		lineEnd := strings.IndexByte(rest[offset:], '\n')
		line := rest[offset:]
		next := len(rest)
		if lineEnd != -1 {
			line = rest[offset : offset+lineEnd]
			next = offset + lineEnd + 1
		}

		if strings.TrimRight(line, " \t") == delimiter {
			var meta struct {
				Subcategory string `yaml:"subcategory"`
			}
			// Tolerate malformed YAML: strip the block and leave the subcategory empty, returning the error to log.
			warn = yaml.Unmarshal([]byte(rest[:offset]), &meta)
			subcategory = sanitize.TruncateToValidUTF8(strings.TrimSpace(meta.Subcategory), maxProviderDocSubcategoryLength)
			return strings.TrimLeft(rest[next:], "\n"), subcategory, warn
		}

		if lineEnd == -1 {
			break
		}
		offset = next
	}

	return normalized, "", nil
}
