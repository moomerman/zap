package server

import "testing"

func TestEnv(t *testing.T) {
	env, err := readEnvFile("../rails/test/rails5.1")
	if err != nil {
		t.Error(err)
	}

	if env[0] != "MOO=foo" {
		t.Error("expected env var to match")
	}
}
