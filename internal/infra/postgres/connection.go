package postgres

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/labib0x9/ProjectUnsafe/config"
	_ "github.com/lib/pq"
)

type PostgreSQL struct {
	*sqlx.DB
}

func New() *PostgreSQL {
	return &PostgreSQL{}
}

func newConnectionString(cfg *config.DbConfig) string {
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s", cfg.DBUser, cfg.DBPass, cfg.DBAddr, cfg.DBPort, cfg.DBName, cfg.DBSslMode)
}

func (p *PostgreSQL) NewConnection(cfg *config.DbConfig) (*sqlx.DB, error) {
	dbSource := newConnectionString(cfg)
	return sqlx.Connect("postgres", dbSource)
}

func newSuperConnectionString(cfg *config.DbConfig) string {
	return fmt.Sprintf("user=%s password= host=%s port=%s dbname=%s sslmode=%s", cfg.DBSuperUser, cfg.DBAddr, cfg.DBPort, cfg.DBSuperDB, cfg.DBSslMode)
}

func (p *PostgreSQL) NewSuperConnection(cfg *config.DbConfig) (*sqlx.DB, error) {
	dbSource := newSuperConnectionString(cfg)
	return sqlx.Connect("postgres", dbSource)
}

func (p *PostgreSQL) Setup(cnf *config.DbConfig) error {
	superDb, err := p.NewSuperConnection(cnf)
	if err != nil {
		return err
	}
	defer superDb.Close()

	_, err = superDb.Exec(fmt.Sprintf(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '%s') THEN
				CREATE ROLE %s WITH LOGIN PASSWORD '%s';
			END IF;
		END
		$$;
	`, cnf.DBUser, cnf.DBUser, cnf.DBPass))
	if err != nil {
		return err
	}

	var exists bool
	err = superDb.QueryRow(
		`SELECT EXISTS(SELECT FROM pg_database WHERE datname = $1)`,
		cnf.DBName,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = superDb.Exec(fmt.Sprintf(
			`CREATE DATABASE %s OWNER %s`,
			cnf.DBName, cnf.DBUser,
		))
		if err != nil {
			return err
		}
	}

	_, err = superDb.Exec(fmt.Sprintf(
		`GRANT ALL PRIVILEGES ON DATABASE %s TO %s`,
		cnf.DBName, cnf.DBUser,
	))
	if err != nil {
		return err
	}

	if err := p.runMigrations(cnf); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	fmt.Println("Database setup complete")

	return nil
}

func (p *PostgreSQL) runMigrations(cnf *config.DbConfig) error {
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
func (p *PostgreSQL) RollbackMigrations(cnf *config.DbConfig, steps int) error {
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

func (p *PostgreSQL) SetupAndConnection(cnf *config.DbConfig) *sqlx.DB {
	if err := p.Setup(cnf); err != nil {
		panic(err)
	}

	dbConn, err := p.NewConnection(cnf)
	if err != nil {
		panic(err)
	}

	return dbConn
}
