// Package postgres provides a Gnomock Preset for PostgreSQL database
package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // postgres driver
	"github.com/orlangure/gnomock"
)

const defaultUser = "postgres"
const defaultPassword = "password"
const defaultDatabase = "postgres"
const defaultSSLMode = "disable"
const defaultPort = 5432

// Preset creates a new Gmomock Postgres preset. This preset includes a Postgres
// specific healthcheck function, default Postgres image and port, and allows to
// optionally set up initial state
func Preset(opts ...Option) gnomock.Preset {
	p := &P{
		DB: defaultDatabase,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// P is a Gnomock Preset implementation of PostgreSQL database
type P struct {
	DB       string
	Queries  []string
	User     string
	Password string
}

// Image returns an image that should be pulled to create this container
func (p *P) Image() string {
	return "docker.io/library/postgres"
}

// Ports returns ports that should be used to access this container
func (p *P) Ports() gnomock.NamedPorts {
	return gnomock.DefaultTCP(defaultPort)
}

// Options returns a list of options to configure this container
func (p *P) Options() []gnomock.Option {
	if p.User != "" && p.Password != "" {
		q := fmt.Sprintf(
			`create user %s with superuser password '%s'`,
			p.User, p.Password,
		)

		p.Queries = append(p.Queries, q)
	}

	opts := []gnomock.Option{
		gnomock.WithHealthCheck(p.healthcheck),
		gnomock.WithEnv("POSTGRES_PASSWORD=" + defaultPassword),
		gnomock.WithInit(p.initf(p.Queries)),
	}

	return opts
}

func (p *P) healthcheck(c *gnomock.Container) error {
	db, err := connect(c, defaultDatabase)
	if err != nil {
		return err
	}

	var one int

	row := db.QueryRow(`select 1`)

	err = row.Scan(&one)
	if err != nil {
		return err
	}

	if one != 1 {
		return fmt.Errorf("unexpected healthcheck result: 1 != %d", one)
	}

	return nil
}

func (p *P) initf(queries []string) gnomock.InitFunc {
	return func(c *gnomock.Container) error {
		if p.DB != defaultDatabase {
			db, err := connect(c, defaultDatabase)
			if err != nil {
				return err
			}

			_, err = db.Exec("create database " + p.DB)
			if err != nil {
				return err
			}
		}

		db, err := connect(c, p.DB)
		if err != nil {
			return err
		}

		for _, q := range queries {
			_, err = db.Exec(q)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func connect(c *gnomock.Container, db string) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port(gnomock.DefaultPort),
		defaultUser, defaultPassword, db, defaultSSLMode,
	)

	return sql.Open("postgres", connStr)
}
