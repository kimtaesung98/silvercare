// Command admin does operator tasks against the database in DATABASE_URL:
//
//	admin create-tablet -elder <elder id> [-label 거실 태블릿]   prints a new tablet token
//	admin set-caregiver-login -caregiver <id> -login <login id>  reads the password from stdin
//	admin seed-demo                                             adds a demo center, elder, caregiver, visit and tablet
//
// Tokens and passwords are printed once and only their hashes are stored.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/auth"
	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "admin:", err)
		os.Exit(1)
	}
}

const usage = "usage: admin create-tablet | set-caregiver-login | seed-demo (see -h of each)"

func run(ctx context.Context, args []string, stdin io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	switch args[0] {
	case "create-tablet":
		return createTablet(ctx, pool, args[1:], out)
	case "set-caregiver-login":
		return setCaregiverLogin(ctx, pool, args[1:], stdin, out)
	case "seed-demo":
		return seedDemo(ctx, pool, out)
	default:
		return errors.New(usage)
	}
}

func createTablet(ctx context.Context, pool *pgxpool.Pool, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("create-tablet", flag.ContinueOnError)
	elder := fs.String("elder", "", "elder id")
	label := fs.String("label", "", "device label")
	if err := fs.Parse(args); err != nil {
		return err
	}
	elderID, err := uuid.Parse(*elder)
	if err != nil {
		return fmt.Errorf("-elder: %w", err)
	}
	token, err := newTablet(ctx, db.New(pool), elderID, *label)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "tablet token (shown once): %s\n", token)
	return nil
}

func newTablet(ctx context.Context, q *db.Queries, elderID uuid.UUID, label string) (string, error) {
	token := auth.NewDeviceToken()
	var l *string
	if label != "" {
		l = &label
	}
	if _, err := q.CreateElderTablet(ctx, db.CreateElderTabletParams{
		ElderID: &elderID, TokenHash: auth.HashDeviceToken(token), Label: l,
	}); err != nil {
		return "", fmt.Errorf("create tablet: %w", err)
	}
	return token, nil
}

func setCaregiverLogin(ctx context.Context, pool *pgxpool.Pool, args []string, stdin io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("set-caregiver-login", flag.ContinueOnError)
	caregiver := fs.String("caregiver", "", "caregiver id")
	login := fs.String("login", "", "login id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id, err := uuid.Parse(*caregiver)
	if err != nil {
		return fmt.Errorf("-caregiver: %w", err)
	}
	if *login == "" {
		return errors.New("-login is required")
	}
	fmt.Fprint(out, "password: ")
	pw, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	pw = strings.TrimRight(pw, "\r\n")
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if err := setLogin(ctx, db.New(pool), id, *login, pw); err != nil {
		return err
	}
	fmt.Fprintf(out, "\nlogin set for caregiver %s\n", id)
	return nil
}

func setLogin(ctx context.Context, q *db.Queries, id uuid.UUID, login, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = q.SetCaregiverLogin(ctx, db.SetCaregiverLoginParams{ID: id, LoginID: &login, PasswordHash: &hash})
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("caregiver %s not found", id)
	}
	return err
}

// seedDemo adds one center with an elder, a caregiver who can log in, a visit
// 30 minutes from now and the elder's tablet, for trying the API locally.
func seedDemo(ctx context.Context, pool *pgxpool.Pool, out io.Writer) error {
	login := "demo-" + uuid.NewString()[:6]
	password := uuid.NewString()[:12]
	var token string
	var elderID, caregiverID, visitID uuid.UUID
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			WITH g AS (INSERT INTO guardian (name, phone) VALUES ('데모 보호자', '010-0000-0000') RETURNING id),
			     c AS (INSERT INTO daycare_center (name) VALUES ('데모 주간보호센터') RETURNING id),
			     e AS (INSERT INTO elder (name, birth_date, dementia_stage, guardian_id, center_id)
			           SELECT '김순자', '1940-03-01', 'MILD', g.id, c.id FROM g, c RETURNING id, center_id),
			     cg AS (INSERT INTO caregiver (name, center_id) SELECT '이조무', e.center_id FROM e RETURNING id),
			     v AS (INSERT INTO visit (elder_id, caregiver_id, scheduled_time)
			           SELECT e.id, cg.id, $1 FROM e, cg RETURNING id)
			SELECT e.id, cg.id, v.id FROM e, cg, v`, time.Now().Add(30*time.Minute),
		).Scan(&elderID, &caregiverID, &visitID)
		if err != nil {
			return err
		}
		q := db.New(tx)
		if err := setLogin(ctx, q, caregiverID, login, password); err != nil {
			return err
		}
		token, err = newTablet(ctx, q, elderID, "데모 태블릿")
		return err
	})
	if err != nil {
		return fmt.Errorf("seed demo: %w", err)
	}
	fmt.Fprintf(out, "caregiver login: %s / %s\nelder: %s\nvisit (in 30 min): %s\ntablet token: %s\n",
		login, password, elderID, visitID, token)
	return nil
}
