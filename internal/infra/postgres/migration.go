package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/labib0x9/ffgif/config"
)

func SetupDatabase(cnf *config.PostgreSQL) error {
	superDbConn := NewPostgresSuperConn(cnf)
	defer superDbConn.Close()

	_, err := superDbConn.Exec(fmt.Sprintf(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '%s') THEN
				CREATE ROLE %s WITH LOGIN PASSWORD '%s';
			END IF;
		END
		$$;
	`, cnf.User, cnf.User, cnf.Pass))
	if err != nil {
		return err
	}

	var exists bool
	err = superDbConn.QueryRow(
		`SELECT EXISTS(SELECT FROM pg_database WHERE datname = $1)`,
		cnf.DatabaseName,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = superDbConn.Exec(fmt.Sprintf(
			`CREATE DATABASE %s OWNER %s`,
			cnf.DatabaseName, cnf.User,
		))
		if err != nil {
			return err
		}
	}

	_, err = superDbConn.Exec(fmt.Sprintf(
		`GRANT ALL PRIVILEGES ON DATABASE %s TO %s`,
		cnf.DatabaseName, cnf.User,
	))
	if err != nil {
		return err
	}

	slog.Info("Database setup complete, run migration to create tables")

	return nil
}

func Run(cnf *config.PostgreSQL) error {
	dbSource := newConnectionString(cnf)

	appDB, err := sql.Open("postgres", dbSource)
	if err != nil {
		return err
	}
	defer appDB.Close()

	driver, err := postgres.WithInstance(appDB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

// step = 0, all
func Rollback(cnf *config.PostgreSQL, steps int) error {
	dbSource := newConnectionString(cnf)
	appDB, err := sql.Open("postgres", dbSource)
	if err != nil {
		return err
	}
	defer appDB.Close()

	driver, err := postgres.WithInstance(appDB, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if steps == 0 {
		return m.Down()
	}
	return m.Steps(-steps)
}
