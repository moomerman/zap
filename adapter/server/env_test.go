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

func TestSanitiseEnvPair(t *testing.T) {
	tests := map[string]string{
		"SECRET_KEY_BASE=abc123":           "SECRET_KEY_BASE=[redacted]",
		"DATABASE_URL=postgres://u:p@h/db": "DATABASE_URL=[redacted]",
		"TOKEN=a=b":                        "TOKEN=[redacted]",
		"RAILS_ENV=development":            "RAILS_ENV=development",
		"PORT=3000":                        "PORT=3000",
		"EMPTY=":                           "EMPTY=",
		"not a pair":                       "[redacted]",
	}

	for in, want := range tests {
		if got := sanitiseEnvPair(in); got != want {
			t.Errorf("sanitiseEnvPair(%q) = %q, want %q", in, got, want)
		}
	}
}
