package server

import "testing"

func TestEnv(t *testing.T) {
	env, err := readEnvFile("testdata/envapp")
	if err != nil {
		t.Fatal(err)
	}

	if len(env) != 1 || env[0] != "MOO=foo" {
		t.Errorf("expected [MOO=foo], got %v", env)
	}
}

func TestEnvMissingFile(t *testing.T) {
	env, err := readEnvFile(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if len(env) != 0 {
		t.Errorf("expected no env vars, got %v", env)
	}
}
