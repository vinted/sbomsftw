package github

import (
	"context"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bradleyfalzon/ghinstallation/v2"
)

func GenerateGithubAppTokenInternal(org string) string {
	appID, installID, privateKeyB64 := fetchEnvVars(org)

	key, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		log.Fatalf("Failed to decode private key: %v", err)
	}

	itr, err := ghinstallation.New(http.DefaultTransport, appID, installID, key)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	token, err := itr.Token(ctx)
	if err != nil {
		log.Fatalf("unable to get github token: %s", err)
	}

	return token
}

func fetchEnvVars(org string) (int64, int64, string) {
	var prefix string
	if org != "" {
		prefix = strings.ToUpper(org) + "_"
	}

	appID := getInt64EnvVar(prefix + "APP_ID")
	installID := getInt64EnvVar(prefix + "INSTALLATION_ID")
	privateKeyB64 := os.Getenv(prefix + "GH_APP_KEY")

	if privateKeyB64 == "" {
		log.Printf("Environment variable %sGH_APP_KEY not set", prefix)
	}

	return appID, installID, privateKeyB64
}

func getInt64EnvVar(envVar string) int64 {
	value := os.Getenv(envVar)
	if value == "" {
		log.Printf("Environment variable %s not set", envVar)
		return -1 // Or any other sentinel value to indicate missing variable
	}

	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Printf("Failed to convert %s to int64: %v", envVar, err)
		return 0 // Or any other sentinel value to indicate parsing failure
	}
	return parsedValue
}
