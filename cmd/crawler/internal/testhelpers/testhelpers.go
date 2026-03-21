package testhelpers

import "testing"

func SetTestEnvs(t *testing.T) {
	t.Helper() // marks this as a helper so failures point to the caller
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "myuser")
	t.Setenv("DB_PASSWORD", "mypassword")
	t.Setenv("DB_NAME", "jobs_webcrawler")
	t.Setenv("DB_SSLMODE", "disable")
}

// #nosec G101 -- this is test code, no real credentials
const ExpectedConnStr = "postgres://myuser:mypassword@localhost:5433/jobs_webcrawler?sslmode=disable"
