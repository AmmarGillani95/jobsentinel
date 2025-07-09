package app

import (
	"database/sql"
	"log"
	"os"

	"github.com/AmmarGillani95/jobsentinel/internal/store"
	"github.com/AmmarGillani95/jobsentinel/migrations"
)

type Application struct {
	Logger *log.Logger
	DB     *sql.DB
}

func NewApplication() (*Application, error) {
	pgDB, err := store.Open()
	if err != nil {
		return nil, err
	}

	err = store.MigrateFS(pgDB, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	app := &Application{
		Logger: logger,
		DB:     pgDB,
	}

	return app, nil
}
