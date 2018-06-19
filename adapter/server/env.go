package server

import (
	"bufio"
	"os"
)

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
