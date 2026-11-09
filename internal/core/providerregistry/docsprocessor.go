package providerregistry

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"golang.org/x/sync/errgroup"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/sanitize"
)

//go:generate go tool mockery --name DocsProcessor --inpackage --case underscore

// maxDocsProcessingErrorMessageLength caps the stored processing error message.
const maxDocsProcessingErrorMessageLength = 2048

// maxConcurrentDocFileUploads bounds concurrent doc file uploads, capping both parallelism and the file
// content held in memory at once.
const maxConcurrentDocFileUploads = 16

// DocsProcessor extracts an uploaded provider-docs package and records its terminal processing status.
type DocsProcessor interface {
	// ProcessProviderVersionDocs processes a version's uploaded docs package. finalAttempt is true on the
	// queue's last delivery, so a version that can't settle is recorded as failed instead of left pending.
	ProcessProviderVersionDocs(ctx context.Context, providerVersionID string, finalAttempt bool) error
}

type docsProcessor struct {
	dbClient *db.Client
	store    RegistryStore
	logger   logger.Logger
}

// NewDocsProcessor returns a DocsProcessor backed by the given dependencies.
func NewDocsProcessor(dbClient *db.Client, store RegistryStore, logger logger.Logger) DocsProcessor {
	return &docsProcessor{dbClient: dbClient, store: store, logger: logger}
}

// ProcessProviderVersionDocs processes a version's uploaded docs package and records its terminal status.
func (p *docsProcessor) ProcessProviderVersionDocs(ctx context.Context, providerVersionID string, finalAttempt bool) error {
	ctx, span := tracer.Start(ctx, "core.ProcessProviderVersionDocs")
	defer span.End()

	err := p.processDocs(ctx, providerVersionID)
	if err == nil {
		return nil
	}

	// A racing delivery already recorded a result, or the version was deleted while processing: ack.
	if settled, sErr := p.isSettled(ctx, providerVersionID); sErr == nil && settled {
		return nil
	}

	// Upload is one-shot, so record a terminal failed status on the last delivery rather than leaving the
	// version stuck pending once the queue stops redelivering.
	if finalAttempt {
		p.logger.WithContextFields(ctx).Errorf("provider docs processing retries exhausted: %v", err)
		return p.recordResult(ctx, providerVersionID, nil,
			errors.New("docs processing retries exhausted", errors.WithErrorCode(errors.EInternal)))
	}

	return errors.Wrap(err, "failed to process provider version docs", errors.WithSpan(span))
}

// processDocs downloads, parses, and uploads the docs package, then records the result. Download and upload
// errors are transient and returned for redelivery; parse errors are terminal and recorded as failed.
func (p *docsProcessor) processDocs(ctx context.Context, providerVersionID string) error {
	providerVersion, err := p.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil
		}
		return errors.Wrap(err, "failed to get provider version by ID")
	}

	provider, err := p.getProviderByID(ctx, providerVersion.ProviderID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil
		}
		return errors.Wrap(err, "failed to get provider by ID")
	}

	pkg, err := p.downloadDocsPackage(ctx, provider, providerVersion)
	if err != nil {
		return err
	}

	// Stream the package: parse and upload each doc file as it is read so the whole decompressed package is
	// never held in memory. A parse error is deterministic and terminal, so it is recorded as failed.
	uploaded, terminal, err := p.uploadDocFiles(ctx, provider, providerVersion, pkg)
	if err != nil {
		if terminal {
			return p.recordResult(ctx, providerVersionID, nil, err)
		}
		return err
	}
	if len(uploaded) == 0 {
		return p.recordResult(ctx, providerVersionID, nil,
			errors.New("no documentation files found; expected an index.md at the root or Markdown files under resources/, data-sources/, ephemeral-resources/, functions/, or guides/", errors.WithErrorCode(errors.EInvalid)))
	}

	return p.recordResult(ctx, providerVersionID, uploaded, nil)
}

// downloadDocsPackage reads the uploaded package fully into memory, bounded by the maxDocsUploadSize cap on
// upload, so download I/O stays separate from deterministic in-memory parsing.
func (p *docsProcessor) downloadDocsPackage(ctx context.Context, provider *models.TerraformProvider, providerVersion *models.TerraformProviderVersion) ([]byte, error) {
	reader, err := p.store.GetProviderVersionDocsPackage(ctx, providerVersion, provider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download docs package")
	}
	defer reader.Close()

	pkg, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read docs package")
	}

	return pkg, nil
}

// uploadDocFiles validates, uploads, and links each parsed doc file, returning the validated doc items;
// terminal is true for a bad-content failure and false for a transient upload or link failure.
func (p *docsProcessor) uploadDocFiles(ctx context.Context, provider *models.TerraformProvider, providerVersion *models.TerraformProviderVersion, pkg []byte) (uploaded []*models.TerraformProviderDocItem, terminal bool, err error) {
	reader, err := newBoundedTarReader(bytes.NewReader(pkg))
	if err != nil {
		return nil, true, err
	}
	defer reader.close()

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentDocFileUploads)

	uploaded = []*models.TerraformProviderDocItem{}
	seen := map[string]struct{}{}
	for {
		docFile, content, warn, err := nextDocFile(reader, seen)
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = g.Wait()
			return nil, true, err
		}
		if warn != nil {
			p.logger.WithContextFields(ctx).Warnf("ignoring malformed front-matter in doc file %s: %v", docFile.Name, warn)
		}

		// A missing subcategory is stored as NULL so ungrouped doc items are distinct from an empty group.
		var subcategory *string
		if docFile.Subcategory != "" {
			subcategory = &docFile.Subcategory
		}

		docItem := &models.TerraformProviderDocItem{
			ProviderVersionID: providerVersion.Metadata.ID,
			Filename:          docFile.Name,
			Category:          docFile.Category,
			Subcategory:       subcategory,
		}

		if err = docItem.Validate(); err != nil {
			_ = g.Wait()
			return nil, true, errors.Wrap(err, "invalid doc file %s", docFile.Name)
		}

		uploaded = append(uploaded, docItem)
		g.Go(func() error {
			retainFn, _, err := p.store.UploadProviderVersionDocFile(gCtx, providerVersion, provider, docItem.Category, docItem.Filename, strings.NewReader(content))
			if err != nil {
				return errors.Wrap(err, "failed to upload doc file %s", docItem.Filename)
			}

			if err = retainFn(gCtx, providerVersion.Metadata.ID); err != nil {
				return errors.Wrap(err, "failed to link doc file object store ref %s", docItem.Filename)
			}
			return nil
		})
	}

	if err = g.Wait(); err != nil {
		return nil, false, err
	}

	return uploaded, false, nil
}

// recordResult writes the catalog rows on success and the terminal status in one transaction, skipping the
// write when the version is no longer pending. Doc file object refs are already linked during upload.
func (p *docsProcessor) recordResult(ctx context.Context, providerVersionID string, uploaded []*models.TerraformProviderDocItem, processErr error) error {
	txCtx, err := p.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin docs result transaction")
	}
	defer func() {
		if rErr := p.dbClient.Transactions.RollbackTx(txCtx); rErr != nil {
			p.logger.WithContextFields(ctx).Errorf("failed to roll back docs result transaction: %v", rErr)
		}
	}()

	providerVersion, err := p.getProviderVersionByID(txCtx, providerVersionID)
	if err != nil {
		// The version was deleted before the result could be recorded; nothing to do.
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil
		}
		return err
	}

	// Another delivery already recorded a result, so this one is a no-op.
	if providerVersion.DocsProcessingStatus != models.TerraformProviderVersionDocsProcessingStatusPending {
		return nil
	}

	if processErr == nil {
		if err = p.dbClient.TerraformProviderDocItems.CreateProviderDocItems(txCtx, uploaded); err != nil {
			return errors.Wrap(err, "failed to create doc items")
		}

		providerVersion.DocsProcessingStatus = models.TerraformProviderVersionDocsProcessingStatusCompleted
	} else {
		providerVersion.DocsProcessingStatus = models.TerraformProviderVersionDocsProcessingStatusFailed
		providerVersion.DocsProcessingErrorMessage = ptr.String(
			sanitize.TruncateToValidUTF8(errors.ErrorMessage(processErr), maxDocsProcessingErrorMessageLength),
		)
	}

	if _, err = p.dbClient.TerraformProviderVersions.UpdateProviderVersion(txCtx, providerVersion); err != nil {
		return errors.Wrap(err, "failed to set provider version docs status to %s", providerVersion.DocsProcessingStatus)
	}

	return p.dbClient.Transactions.CommitTx(txCtx)
}

// isSettled reports whether the version is gone or already has a terminal result.
func (p *docsProcessor) isSettled(ctx context.Context, providerVersionID string) (bool, error) {
	providerVersion, err := p.getProviderVersionByID(ctx, providerVersionID)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return true, nil
		}
		return false, err
	}

	return providerVersion.DocsProcessingStatus != models.TerraformProviderVersionDocsProcessingStatusPending, nil
}

func (p *docsProcessor) getProviderVersionByID(ctx context.Context, id string) (*models.TerraformProviderVersion, error) {
	version, err := p.dbClient.TerraformProviderVersions.GetProviderVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, errors.New("provider version with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}
	return version, nil
}

func (p *docsProcessor) getProviderByID(ctx context.Context, id string) (*models.TerraformProvider, error) {
	provider, err := p.dbClient.TerraformProviders.GetProviderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errors.New("provider with id %s not found", id, errors.WithErrorCode(errors.ENotFound))
	}
	return provider, nil
}
