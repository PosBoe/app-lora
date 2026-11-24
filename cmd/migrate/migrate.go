// Package migrate applies the PostgreSQL migrations of cmd/lora/migrations.
// It boots no fx application: it needs only the config and one connection.
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"text/tabwriter"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
	"github.com/urfave/cli/v3"

	"github.com/waneock/app-lora/bootstrap"
	"github.com/waneock/app-lora/cmd/lora/migrations"
)

// defaultDir is where create writes a new migration, relative to the
// repository root, where task runs.
const defaultDir = "cmd/lora/migrations"

// Command returns the migrate command group.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "manage the PostgreSQL migrations",
		Commands: []*cli.Command{
			{
				Name:  "up",
				Usage: "apply every pending migration",
				Action: withProvider(func(ctx context.Context, p *goose.Provider) error {
					results, err := p.Up(ctx)
					printResults(results)
					return err
				}),
			},
			{
				Name:  "down",
				Usage: "roll back the last applied migration",
				Action: withProvider(func(ctx context.Context, p *goose.Provider) error {
					result, err := p.Down(ctx)
					if result != nil {
						printResults([]*goose.MigrationResult{result})
					}
					return err
				}),
			},
			{
				Name:  "status",
				Usage: "show the state of every migration",
				Action: withProvider(func(ctx context.Context, p *goose.Provider) error {
					statuses, err := p.Status(ctx)
					if err != nil {
						return err
					}
					w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
					fmt.Fprintln(w, "VERSION\tSTATE\tAPPLIED AT\tFILE")
					for _, s := range statuses {
						applied := "-"
						if !s.AppliedAt.IsZero() {
							applied = s.AppliedAt.Format(time.DateTime)
						}
						fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Source.Version, s.State, applied, s.Source.Path)
					}
					return w.Flush()
				}),
			},
			{
				Name:      "create",
				Usage:     "create an empty SQL migration named with the current time",
				ArgsUsage: "<name>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "dir", Value: defaultDir, Usage: "write the migration into `DIR`"},
				},
				Action: create,
			},
		},
	}
}

func withProvider(fn func(context.Context, *goose.Provider) error) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		conf, err := bootstrap.LoadConfig(cmd)
		if err != nil {
			return err
		}

		db, err := sql.Open("pgx", conf.PG.DSN)
		if err != nil {
			return err
		}
		defer func() { _ = db.Close() }()

		p, err := newProvider(db, migrations.FS)
		if err != nil {
			return err
		}
		return fn(ctx, p)
	}
}

func newProvider(db *sql.DB, fsys fs.FS) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, db, fsys)
}

func printResults(results []*goose.MigrationResult) {
	if len(results) == 0 {
		fmt.Println("no migrations to run")
		return
	}
	for _, r := range results {
		fmt.Println(r)
	}
}

var nameRE = regexp.MustCompile(`^[a-z0-9]+(_[a-z0-9]+)*$`)

func create(_ context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return errors.New("migrate create: name is required, e.g. create add_users")
	}
	if !nameRE.MatchString(name) {
		return fmt.Errorf("migrate create: name %q must be snake_case", name)
	}

	// UTC, so the versions of two developers in two time zones sort in the
	// order the files were written.
	version := time.Now().UTC().Format("20060102150405")
	path := fmt.Sprintf("%s/%s_%s.sql", cmd.String("dir"), version, name)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("migrate create: %s exists", path)
	}

	const body = "-- +goose Up\n\n-- +goose Down\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}
