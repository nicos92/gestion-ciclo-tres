package sqlite

import (
	"context"
	"database/sql"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestSeedCreatesExampleData(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := Migrate(ctx, db, realMigrationsFS(t)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := Seed(ctx, db); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	assertUser(t, ctx, db, adminUser, adminPassword, 1)
	assertUser(t, ctx, db, produccionUser, produccionPassword, 4)
}

func TestSeedIdempotent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	tfs := realMigrationsFS(t)

	if err := Migrate(ctx, db, tfs); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := Seed(ctx, db); err != nil {
		t.Fatalf("Seed 1ra vez: %v", err)
	}
	if err := Seed(ctx, db); err != nil {
		t.Fatalf("Seed 2da vez: %v", err)
	}

	assertCount(t, ctx, db, "SELECT COUNT(*) FROM usuarios", 2)
}

func TestSeedErrorIfRolesAreNotSeeded(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := Migrate(ctx, db, realMigrationsFS(t)); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM roles`); err != nil {
		t.Fatalf("vaciar roles: %v", err)
	}
	if err := Seed(ctx, db); err == nil {
		t.Fatal("Seed sin roles seedados: se esperaba error")
	}
}

func assertUser(t *testing.T, ctx context.Context, db *sql.DB, username, password string, rolWant int64) {
	t.Helper()
	var hash string
	var rol int64
	if err := db.QueryRowContext(ctx,
		`SELECT password, id_rol FROM usuarios WHERE username=?`, username).Scan(&hash, &rol); err != nil {
		t.Fatalf("usuario %s: %v", username, err)
	}
	if rol != rolWant {
		t.Errorf("usuario %s: id_rol = %d, want %d", username, rol, rolWant)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Errorf("password %s no valida: %v", username, err)
	}
}

func assertCount(t *testing.T, ctx context.Context, db *sql.DB, query string, want int) {
	t.Helper()
	var n int
	if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	if n != want {
		t.Errorf("%s = %d, want %d", query, n, want)
	}
}
