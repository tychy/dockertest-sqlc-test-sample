package main_test

import (
	"context"
	"dockertest-sqlc-test-sample/db"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var q *db.Queries
var conn *pgx.Conn

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	pool.MaxWait = 10 * time.Second

	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	pwd, _ := os.Getwd()

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=user_name",
			"POSTGRES_DB=dbname",
			"listen_addresses = '*'",
		},
		Mounts: []string{
			fmt.Sprintf("%s/schema.sql:/docker-entrypoint-initdb.d/schema.sql", pwd),
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	dbPath := fmt.Sprintf("postgres://user_name:secret@%s/dbname?sslmode=disable", resource.GetHostPort("5432/tcp"))
	if err := pool.Retry(func() error {
		conn, err = pgx.Connect(context.Background(), dbPath)
		if err != nil {
			return err
		}

		if conn.Ping(context.Background()); err != nil {
			return err
		}
		q = db.New(conn)
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestUpdateUserAges(t *testing.T) {
	u, err := q.CreateUser(context.Background(), db.CreateUserParams{
		Name:  "test",
		Email: "test@test.com",
		Age:   20,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = q.UpdateUserAges(context.Background(), db.UpdateUserAgesParams{
		ID:  u.ID,
		Age: u.Age + 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	u, err = q.GetUser(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.Age != 21 {
		t.Fatalf("expected age to be 21, got %d", u.Age)
	}
}

func TestUpdateUserAgesWithTransaction(t *testing.T) {
	u, err := q.CreateUser(context.Background(), db.CreateUserParams{
		Name:  "test",
		Email: "test@test.com",
		Age:   20,
	})
	if err != nil {
		t.Fatal(err)
	}

	c := context.Background()
	tx, err := conn.Begin(c)
	if err != nil {
		t.Fatal(err)
	}

	q := q.WithTx(tx)
	u, err = q.GetUser(c, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	err = q.UpdateUserAges(c, db.UpdateUserAgesParams{
		ID:  u.ID,
		Age: u.Age + 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(c); err != nil {
		t.Fatal(err)
	}

	q = db.New(conn)
	u, err = q.GetUser(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.Age != 21 {
		t.Fatalf("expected age to be 21, got %d", u.Age)
	}
}
