package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres" // Register Postgres dialect
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const initialResourceVersion int = 1

// Key type is used for attaching state to the context
type key string

func (k key) String() string {
	return fmt.Sprintf("gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db/dbclient %s", string(k))
}

const (
	txKey key = "tx"
)

var (
	// ErrOptimisticLockError is used for optimistic lock exceptions
	ErrOptimisticLockError = te.New(
		te.EOptimisticLock,
		"resource version does not match specified version",
	)
	// ErrInvalidID is used for invalid resource UUIDs
	ErrInvalidID = te.New(
		te.EInvalid,
		"invalid id: the id must be a valid uuid",
	)
)

var (
	metadataFieldList = []interface{}{"id", "created_at", "updated_at", "version"}
	dialect           = goqu.Dialect("postgres")
)

// connection is used to represent a DB connection
type connection interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}

// Client acts as a facade for the database
type Client struct {
	conn                        *pgxpool.Pool
	logger                      logger.Logger
	Events                      Events
	Groups                      Groups
	Runs                        Runs
	Jobs                        Jobs
	Plans                       Plans
	Applies                     Applies
	ConfigurationVersions       ConfigurationVersions
	StateVersionOutputs         StateVersionOutputs
	Workspaces                  Workspaces
	StateVersions               StateVersions
	ManagedIdentities           ManagedIdentities
	ServiceAccounts             ServiceAccounts
	Users                       Users
	NamespaceMemberships        NamespaceMemberships
	Teams                       Teams
	TeamMembers                 TeamMembers
	Transactions                Transactions
	Variables                   Variables
	TerraformProviders          TerraformProviders
	TerraformProviderVersions   TerraformProviderVersions
	TerraformProviderPlatforms  TerraformProviderPlatforms
	TerraformModules            TerraformModules
	TerraformModuleVersions     TerraformModuleVersions
	TerraformModuleAttestations TerraformModuleAttestations
	GPGKeys                     GPGKeys
	SCIMTokens                  SCIMTokens
	VCSProviders                VCSProviders
	WorkspaceVCSProviderLinks   WorkspaceVCSProviderLinks
	ActivityEvents              ActivityEvents
	VCSEvents                   VCSEvents
	Roles                       Roles
	Runners                     Runners
	ResourceLimits              ResourceLimits
}

// NewClient creates a new Client
func NewClient(
	ctx context.Context,
	dbHost string,
	dbPort int,
	dbName string,
	dbSslMode string,
	dbUsername string,
	dbPassword string,
	dbMaxConnections int,
	dbAutoMigrateEnabled bool,
	logger logger.Logger,
) (*Client, error) {
	dbURI := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", dbUsername, dbPassword, dbHost, dbPort, dbName, dbSslMode)

	cfg, err := pgxpool.ParseConfig(dbURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db connection URI: %w", err)
	}

	if dbMaxConnections != 0 {
		cfg.MaxConns = int32(dbMaxConnections)
	}

	logger.Infof("Connecting to DB (host=%s, maxConnections=%d)", dbHost, cfg.MaxConns)

	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		logger.Errorf("Unable to connect to DB: %v\n", err)
		return nil, err
	}

	logger.Infof("Successfully connected to DB %s", dbHost)

	// Auto migrate-up the DB if enabled.
	if dbAutoMigrateEnabled {
		logger.Info("Starting DB migrate")

		migrations, err := newMigrations(logger, cfg.ConnString())
		if err != nil {
			return nil, err
		}

		err = migrations.migrateUp()
		if err == migrate.ErrNoChange {
			logger.Info("No migration necessary since DB is already on latest version")
		} else if err != nil {
			logger.Errorf("Unable to migrate DB: %v", err)
			return nil, err
		} else {
			logger.Info("Successfully migrated DB to latest version")
		}
	}

	dbClient := &Client{
		conn:   pool,
		logger: logger,
	}

	dbClient.Events = NewEvents(dbClient)
	dbClient.Groups = NewGroups(dbClient)
	dbClient.Runs = NewRuns(dbClient)
	dbClient.Jobs = NewJobs(dbClient)
	dbClient.Plans = NewPlans(dbClient)
	dbClient.Applies = NewApplies(dbClient)
	dbClient.ConfigurationVersions = NewConfigurationVersions(dbClient)
	dbClient.StateVersionOutputs = NewStateVersionOutputs(dbClient)
	dbClient.Workspaces = NewWorkspaces(dbClient)
	dbClient.StateVersions = NewStateVersions(dbClient)
	dbClient.ManagedIdentities = NewManagedIdentities(dbClient)
	dbClient.ServiceAccounts = NewServiceAccounts(dbClient)
	dbClient.Users = NewUsers(dbClient)
	dbClient.NamespaceMemberships = NewNamespaceMemberships(dbClient)
	dbClient.Teams = NewTeams(dbClient)
	dbClient.TeamMembers = NewTeamMembers(dbClient)
	dbClient.Transactions = NewTransactions(dbClient)
	dbClient.Variables = NewVariables(dbClient)
	dbClient.TerraformProviders = NewTerraformProviders(dbClient)
	dbClient.TerraformProviderVersions = NewTerraformProviderVersions(dbClient)
	dbClient.TerraformProviderPlatforms = NewTerraformProviderPlatforms(dbClient)
	dbClient.TerraformModules = NewTerraformModules(dbClient)
	dbClient.TerraformModuleVersions = NewTerraformModuleVersions(dbClient)
	dbClient.TerraformModuleAttestations = NewTerraformModuleAttestations(dbClient)
	dbClient.GPGKeys = NewGPGKeys(dbClient)
	dbClient.SCIMTokens = NewSCIMTokens(dbClient)
	dbClient.VCSProviders = NewVCSProviders(dbClient)
	dbClient.WorkspaceVCSProviderLinks = NewWorkspaceVCSProviderLinks(dbClient)
	dbClient.ActivityEvents = NewActivityEvents(dbClient)
	dbClient.VCSEvents = NewVCSEvents(dbClient)
	dbClient.Roles = NewRoles(dbClient)
	dbClient.Runners = NewRunners(dbClient)
	dbClient.ResourceLimits = NewResourceLimits(dbClient)

	return dbClient, nil
}

// Close will close the database connections
func (db *Client) Close(_ context.Context) {
	db.conn.Close()
}

func (db *Client) getConnection(ctx context.Context) connection {
	trx, ok := ctx.Value(txKey).(pgx.Tx)
	if !ok {
		// Return a normal DB connection if no transaction exists
		return db.conn
	}
	// Return transaction if it exists on the context
	return trx
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func isUniqueViolation(pgErr *pgconn.PgError) bool {
	return pgErr.Code == pgerrcode.UniqueViolation
}

func isForeignKeyViolation(pgErr *pgconn.PgError) bool {
	return pgErr.Code == pgerrcode.ForeignKeyViolation
}

func isInvalidIDViolation(pgErr *pgconn.PgError) bool {
	return pgErr.Code == pgerrcode.InvalidTextRepresentation && pgErr.Routine == "string_to_uuid"
}

func asPgError(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	ok := errors.As(err, &pgErr)
	if ok {
		return pgErr
	}
	return nil
}

func newResourceID() string {
	return uuid.New().String()
}

func nullableString(val string) sql.NullString {
	return sql.NullString{
		String: val,
		Valid:  val != "",
	}
}

// Produce a rounded version of current time suitable for storing in the DB.
// Because time.Now().UTC() returns nanosecond precision but the DB stores only
// microseconds, it is necessary to round the time to the nearest microsecond
// before storing it.
func currentTime() time.Time {
	return time.Now().UTC().Round(time.Microsecond)
}
