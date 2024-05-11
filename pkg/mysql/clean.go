package mysql

import "github.com/DataDog/datadog-agent/pkg/obfuscate"

func cleanQuery(query string) (string, error) {
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
