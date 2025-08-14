package parser

import (
	"bufio"
	"os"
	"strings"
)

// Example .better-env file:
// # This is a comment
// AWS_API_KEY
// GEMINI_API_KEY
// MONGODB_API_KEY

// ParseBetterEnv reads a .better-env file and returns a list of env vars
func ParseBetterEnvFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var vars []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		vars = append(vars, line)
	}

	return vars, scanner.Err()
}
