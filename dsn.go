package main

import (
	"database/sql"
	"errors"
	"net/url"
	"strings"
)

// ParseDSN parses a Sentry DSN and returns project ID and API key
// DSN format: http://{key}@{host}/{project_id} or https://{key}@{host}/{project_id}
func ParseDSN(dsn string) (projectID string, apiKey string, err error) {
	parsedURL, err := url.Parse(dsn)
	if err != nil {
		return "", "", err
	}

	// Extract API key from user info (format: key@host)
	if parsedURL.User != nil {
		apiKey = parsedURL.User.Username()
		// Also check for password (some DSNs use key:secret format)
		if password, hasPassword := parsedURL.User.Password(); hasPassword && apiKey == "" {
			apiKey = password
		}
	}

	// Extract project ID from path (format: /{project_id})
	path := strings.Trim(parsedURL.Path, "/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) > 0 && pathParts[0] != "" {
		projectID = pathParts[0]
	}

	if apiKey == "" || projectID == "" {
		return "", "", errors.New("invalid DSN format: missing API key or project ID")
	}

	return projectID, apiKey, nil
}

// ValidateProjectAndKey validates that the project ID and API key match
func ValidateProjectAndKey(db *sql.DB, projectID string, apiKey string) error {
	project, err := GetProject(db, projectID)
	if err != nil {
		return err
	}

	if project.APIKey != apiKey {
		return errors.New("invalid API key for project")
	}

	return nil
}