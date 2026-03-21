package dbhelper

import (
	"os"
	"testing"
)

func setTestEnvs(t *testing.T) {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_SSLMODE", "disable")
}

func Test_GetConnectionString(t *testing.T) {
	setTestEnvs(t)

	conn, err := GetConnectionString()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if conn != expected {
		t.Errorf("got %s, want %s", conn, expected)
	}
}

func Test_GetConnectionString_MissingEnv(t *testing.T) {
	_, err := GetConnectionString()
	if err == nil {
		t.Error("expected error when env vars are missing")
	}
}

func Test_GetConnectionString_Success(t *testing.T) {
	setTestEnvs(t)

	_, err := GetConnectionString()
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}
}

func Test_GetConnectionString_MissingEnvVars(t *testing.T) {
	vars := []string{"DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT", "DB_NAME", "DB_SSLMODE"}

	for _, v := range vars {
		t.Run("missing_"+v, func(t *testing.T) {
			setTestEnvs(t)
			os.Unsetenv(v)

			_, err := GetConnectionString()
			if err == nil {
				t.Errorf("expected error when %s is missing", v)
			}
		})
	}
}

func Test_GetConnectionString_Format(t *testing.T) {
	setTestEnvs(t)

	connStr, err := GetConnectionString()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable"
	if connStr != expected {
		t.Errorf("expected connection string '%s', got '%s'", expected, connStr)
	}

	_, err = GetConnectionString()
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

}
