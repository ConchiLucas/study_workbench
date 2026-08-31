package initialize

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestOpenConfiguredDatabaseReturnsDB(t *testing.T) {
	t.Parallel()

	expected := &gorm.DB{}
	db, err := openConfiguredDatabase(func() *gorm.DB {
		return expected
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if db != expected {
		t.Fatalf("expected db pointer to be preserved")
	}
}

func TestOpenConfiguredDatabaseRecoversPanic(t *testing.T) {
	t.Parallel()

	db, err := openConfiguredDatabase(func() *gorm.DB {
		panic(errors.New("boom"))
	})
	if err == nil {
		t.Fatal("expected error when open panics")
	}
	if db != nil {
		t.Fatal("expected nil db when open panics")
	}
}
