package common

import (
	"net/url"
	"strings"
)

const (
	DatabaseTypeMySQL      = "mysql"
	DatabaseTypeSQLite     = "sqlite"
	DatabaseTypePostgreSQL = "postgres"
)

var UsingSQLite = false
var UsingPostgreSQL = false
var LogSqlType = DatabaseTypeSQLite
var UsingMySQL = false
var UsingClickHouse = false

var SQLitePath = "one-api.db"

var sqliteDefaultPragmas = []struct {
	name  string
	value string
}{
	{"busy_timeout", "busy_timeout(5000)"},
	{"journal_mode", "journal_mode(WAL)"},
	{"synchronous", "synchronous(NORMAL)"},
	{"mmap_size", "mmap_size(268435456)"},
	{"cache_size", "cache_size(-65536)"},
	{"temp_store", "temp_store(MEMORY)"},
}

func BuildSQLiteDSN(path string) string {
	if path == "" {
		path = "one-api.db"
	}

	query := ""
	if pos := strings.IndexRune(path, '?'); pos >= 0 {
		query = path[pos+1:]
		path = path[:pos]
	}

	values, err := url.ParseQuery(query)
	if err != nil {
		values = url.Values{}
	}

	existing := make(map[string]struct{}, len(values["_pragma"]))
	for _, p := range values["_pragma"] {
		name := strings.TrimSpace(p)
		if i := strings.IndexAny(name, "( \t"); i >= 0 {
			name = name[:i]
		}
		name = strings.ToLower(name)
		existing[name] = struct{}{}
	}

	for _, pg := range sqliteDefaultPragmas {
		if _, ok := existing[pg.name]; ok {
			continue
		}
		values.Add("_pragma", pg.value)
	}

	enc := values.Encode()
	if enc == "" {
		return path
	}
	return path + "?" + enc
}
