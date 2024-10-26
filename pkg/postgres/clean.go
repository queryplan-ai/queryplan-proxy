package postgres

import (
	"fmt"
	"strings"

	"github.com/DataDog/datadog-agent/pkg/obfuscate"
)

func cleanQuery(query string) (string, error) {
	// remove the \x00 control characters from the start and end of the query

	for strings.HasPrefix(query, "\x00") {
		query = query[1:]
	}

	for strings.HasSuffix(query, "\x00") {
		query = query[:len(query)-1]
	}

	query = strings.TrimSpace(query)

	fmt.Printf("Query: %q\n", query)
	cleanedQuery, err := obfuscate.NewObfuscator(obfuscate.Config{
		SQL: obfuscate.SQLConfig{
			KeepSQLAlias: true,
		},
	}).ObfuscateSQLString(query)
	if err != nil {
		return "", err
	}

	return cleanedQuery.Query, nil
}
