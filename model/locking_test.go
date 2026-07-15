package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

// Dummy dialector: real SQLite drivers strip FOR UPDATE from generated SQL.
func TestLockForUpdateEmitsRowLock(t *testing.T) {
	dummyDB, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	require.NoError(t, err)
	buildSQL := func() string {
		var rows []Redemption
		return lockForUpdate(dummyDB).Where("id = ?", 1).Find(&rows).Statement.SQL.String()
	}

	prevSQLite, prevMySQL, prevPG := common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL
	t.Cleanup(func() {
		common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = prevSQLite, prevMySQL, prevPG
	})

	common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = false, true, false
	assert.Contains(t, buildSQL(), "FOR UPDATE")

	common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = false, false, true
	assert.Contains(t, buildSQL(), "FOR UPDATE")

	common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = true, false, false
	assert.NotContains(t, buildSQL(), "FOR UPDATE")
}
