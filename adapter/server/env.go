package server

import (
	"bufio"
	"os"
	"strings"
)

// plainEnvKeys are env vars whose values are safe to show in the log.
var plainEnvKeys = map[string]bool{
	"PORT":      true,
	"HOST":      true,
	"APP_ENV":   true,
	"GO_ENV":    true,
	"MIX_ENV":   true,
	"NODE_ENV":  true,
	"RACK_ENV":  true,
	"RAILS_ENV": true,
}

// sanitiseEnvPair returns a KEY=VALUE pair fit for logging, keeping the key
// but redacting the value unless the key is known to be harmless.
func sanitiseEnvPair(pair string) string {
	key, value, ok := strings.Cut(pair, "=")
	if !ok {
		return "[redacted]"
	}
	if value == "" || plainEnvKeys[strings.TrimSpace(key)] {
		return pair
	}
	return key + "=[redacted]"
}

func readEnvFile(dir string) ([]string, error) {
	file := dir + "/.env"
	env := []string{}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return env, nil
	}

	inFile, err := os.Open(file)
	if err != nil {
		return env, err
	}
	defer inFile.Close()
	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		env = append(env, scanner.Text())
	}

	return env, nil
}
