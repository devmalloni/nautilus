package nautilus

import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFolder embed.FS

func (p *SqlPersister) Migrate(forceVersion *int, databaseName string, config *postgres.Config) error {
	driver, err := postgres.WithInstance(p.db.DB, config)
	if err != nil {
		return err
	}

	d, err := iofs.New(migrationsFolder, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		databaseName,
		driver)
	if err != nil {
		return err
	}

	if forceVersion != nil {
		err = m.Force(*forceVersion)
	} else {
		err = m.Up()
	}
	if err != nil {
		return err
	}

	return nil
}
