package api

import (
	"testing"

	"github.com/jamie-anson/project-beacon-runner/internal/db"
)

func TestNewAPIServer_NoDB(t *testing.T) {
	s := NewAPIServer(nil)
	if s == nil {
		t.Fatal("expected server instance")
	}
	// Core components should be initialized
	if s.golemService == nil || s.executor == nil || s.validator == nil || s.wsHub == nil {
		t.Fatalf("expected core components initialized; got golem:%v exec:%v val:%v hub:%v", s.golemService, s.executor, s.validator, s.wsHub)
	}
	// DB-dependent components should be nil
	if s.jobsSvc != nil || s.jobsRepo != nil || s.execsRepo != nil || s.ipfsRepo != nil || s.transparencyRepo != nil || s.ipfsClient != nil || s.ipfsBundler != nil || s.q != nil {
		t.Fatalf("expected DB-dependent components to be nil when DB is absent")
	}
}

func TestNewAPIServer_WithNilDBField(t *testing.T) {
	database := &db.DB{DB: nil}
	s := NewAPIServer(database)
	if s == nil {
		t.Fatal("expected server instance")
	}
	// Still should not initialize DB-dependent components when DB field is nil
	if s.jobsSvc != nil || s.q != nil {
		t.Fatalf("expected DB-dependent components to be nil when underlying DB is nil")
	}
}
