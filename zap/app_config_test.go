package zap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.test")
	data := "dir: /src/app\ncommand: bin/rails s -p $PORT\nscheme: https\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := readConfigFromFile(path, "app.test")
	if err != nil {
		t.Fatal(err)
	}
	want := AppConfig{Scheme: "https", Host: "app.test", Port: "PORT", Dir: "/src/app", Command: "bin/rails s -p $PORT", Key: "/src/app"}
	if *config != want {
		t.Errorf("expected %+v, got %+v", want, *config)
	}
}

func TestClosestMatchingPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".zap"), 0755); err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(home, ".zap", "app.test")
	if err := os.WriteFile(app, []byte("proxy: http://127.0.0.1:3000\n"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, host := range []string{"app.test", "api.app.test"} {
		path, err := getClosestMatchingPath(host)
		if err != nil {
			t.Fatal(host, err)
		}
		if path != app {
			t.Errorf("%s: expected %s, got %s", host, app, path)
		}
	}
	if _, err := getClosestMatchingPath("other.test"); err == nil {
		t.Error("expected an error for an unknown host")
	}
}
