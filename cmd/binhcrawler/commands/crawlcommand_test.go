package commands

import (
	"bytes"
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func Test_InitDB_SuccessfulConnection(t *testing.T) {
	db, err := InitDb()
	if err != nil {
		t.Errorf("expected no error but got %v", err)
	}
	defer db.Close()
}

func Test_InitDB_Error(t *testing.T) {
	setupDbFnMock := func() (*sql.DB, error) { return nil, fmt.Errorf("Mock Error") }
	setupDatabaseFn = setupDbFnMock

	_, err := InitDb()
	if err == nil {
		t.Errorf("Expected error but not nil")
	}
}

func Test_Execute_DBDeferredClose(t *testing.T) {
	mockDB, _, mockDbErr := sqlmock.New()
	if mockDbErr != nil {
		t.Errorf("unexpected error setting up mock db %v", mockDbErr)
	}

	mockDbSetupFn := func() (*sql.DB, error) {
		return mockDB, nil
	}
	setupDatabaseFn = mockDbSetupFn

	buff := &bytes.Buffer{}
	baseCmd, baseCmdErr := SetupBaseCommand(buff, LogLevelInfo)
	if baseCmdErr != nil {
		t.Errorf("SetupBaseCommand failed with error %v", baseCmdErr)
	}

	cmd := &CrawlCommand{
		BaseCommand: *baseCmd,
		GlobalOpts: GlobalOpts{
			LogLevel: "info",
		},
	}

	_ = cmd.Execute([]string{})

	pingErr := mockDB.Ping()
	if pingErr == nil {
		t.Error("expected error when pinging closed DB, but got nil - Close() was not called")
	}
}
