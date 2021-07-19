package fns_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigRetriever(t *testing.T) {
	path, err := os.Executable()
	if err != nil {
		t.Error(err)
	}
	t.Log(path)
	dir := filepath.Dir(path)
	t.Log(dir)
}
