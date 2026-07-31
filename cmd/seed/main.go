// Command seed generates a Xolo SQLite database populated with fake but
// coherent data, meant to be used as a fixture for end-to-end tests.
//
// The generated dataset is fully deterministic: identifiers, API token values
// and the usage history are derived from a fixed seed, so E2E assertions can
// target stable values (see cmd/seed/fixtures.go for the catalog).
//
// Usage:
//
//	go run ./cmd/seed -dsn e2e.sqlite -force
//	XOLO_STORAGE_DATABASE_DSN=e2e.sqlite XOLO_SECRET_KEY=$(cat e2e.sqlite.key) bin/server
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ncruces/go-sqlite3/gormlite"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	gormadapter "github.com/bornholm/xolo/internal/adapter/gorm"
	"github.com/bornholm/xolo/internal/core/port"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func main() {
	var (
		dsn       = flag.String("dsn", "e2e.sqlite", "path of the SQLite database to generate")
		secretKey = flag.String("secret-key", defaultSecretKey, "32-byte hex key used to encrypt provider API keys (must match XOLO_SECRET_KEY)")
		days      = flag.Int("days", 30, "number of days of usage history to generate")
		randSeed  = flag.Int64("seed", 20260731, "PRNG seed driving the usage history (same seed = same database)")
		force     = flag.Bool("force", false, "delete the database file (and its -wal/-shm siblings) before generating")
		verbose   = flag.Bool("verbose", false, "log every SQL statement")
	)
	flag.Parse()

	ctx := context.Background()

	if err := run(ctx, *dsn, *secretKey, *days, *randSeed, *force, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "seed: %+v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dsn, secretKey string, days int, randSeed int64, force, verbose bool) error {
	if len(secretKey) != 64 {
		return errors.Errorf("secret key must be a 32-byte hex string (64 chars), got %d chars", len(secretKey))
	}

	if force {
		if err := removeDatabaseFiles(dsn); err != nil {
			return errors.WithStack(err)
		}
	} else if exists(dsn) {
		return errors.Errorf("database %q already exists, use -force to overwrite it", dsn)
	}

	logLevel := logger.Error
	if verbose {
		logLevel = logger.Info
	}

	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return errors.WithStack(err)
	}

	internalDB, err := db.DB()
	if err != nil {
		return errors.WithStack(err)
	}
	internalDB.SetMaxOpenConns(1)
	defer internalDB.Close()

	if err := db.Exec("PRAGMA journal_mode=wal; PRAGMA foreign_keys=on; PRAGMA busy_timeout=5000").Error; err != nil {
		return errors.WithStack(err)
	}

	// NewStore lazily runs the gormigrate migrations on its first query: issue a
	// trivial one so the schema exists before we insert anything directly.
	store := gormadapter.NewStore(db)
	if _, err := store.CountUsers(ctx, port.QueryUsersOptions{}); err != nil {
		return errors.Wrap(err, "could not migrate database schema")
	}

	s := &seeder{
		db:        db,
		store:     store,
		dsn:       dsn,
		secretKey: secretKey,
		days:      days,
		rand:      newRand(randSeed),
	}

	if err := s.seed(ctx); err != nil {
		return errors.WithStack(err)
	}

	// Fold the WAL back into the main file so the fixture can be copied around
	// as a single artifact.
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		return errors.WithStack(err)
	}

	s.report()

	return nil
}

func removeDatabaseFiles(dsn string) error {
	base := dsn
	if idx := strings.IndexAny(dsn, "?"); idx >= 0 {
		base = dsn[:idx]
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(base + suffix); err != nil && !os.IsNotExist(err) {
			return errors.WithStack(err)
		}
	}
	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// report prints the catalog of the generated dataset: everything an E2E test
// needs to address the fixture (identifiers, credentials, volumes).
func (s *seeder) report() {
	logf("Base de données générée avec %d enregistrements d'usage sur %d jours.", s.usageRecords, s.days)
	logf("")
	logf("Organisations :")
	logf("  %-12s acme     Acme Corporation    EUR   actif    coût cumulé %s", orgAcme, formatCost(s.usageCost[orgAcme], "EUR"))
	logf("  %-12s globex   Globex Industries   USD   actif    coût cumulé %s", orgGlobex, formatCost(s.usageCost[orgGlobex], "USD"))
	logf("  %-12s initech  Initech             USD   inactif", orgInitech)
	logf("")
	logf("Utilisateurs (provider=local, subject=prénom) :")
	logf("  root@xolo.test    admin plateforme")
	logf("  alice@acme.test   propriétaire Acme")
	logf("  bob@acme.test     administrateur Acme")
	logf("  carol@acme.test   membre Acme + membre Globex")
	logf("  dave@acme.test    membre Acme + rôle « Analyste », quota journalier saturé")
	logf("  erin@globex.test  propriétaire Globex")
	logf("  frank@globex.test membre Globex, compte désactivé")
	logf("")
	logf("Jetons d'API (en clair) :")
	logf("  %-24s alice / acme", tokenAlice)
	logf("  %-24s carol / acme", tokenCarolAcme)
	logf("  %-24s carol / globex", tokenCarolGlobex)
	logf("  %-24s dave / acme (expiré)", tokenDaveExpired)
	logf("  %-24s erin / globex", tokenErin)
	logf("  %-24s application CI (acme)", tokenAppAcmeCI)
	logf("  %-24s application bot (globex, désactivée)", tokenAppGlobex)
	logf("")
	logf("Invitation utilisable : /join/inv-acme-open")
	logf("")
	logf("Lancement du serveur sur cette base :")
	logf("  XOLO_STORAGE_DATABASE_DSN=%s XOLO_SECRET_KEY=%s bin/server", s.dsn, s.secretKey)
}

func formatCost(microcents int64, currency string) string {
	return fmt.Sprintf("%.2f %s", float64(microcents)/1_000_000, currency)
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}
