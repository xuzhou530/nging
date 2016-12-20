package mysql

import (
	"database/sql"

	"strings"

	"github.com/webx-top/com"
	"github.com/webx-top/db/lib/factory"
)

func (m *mySQL) kvVal(sqlStr string) ([]map[string]string, error) {
	r := []map[string]string{}
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var k sql.NullString
		var v sql.NullString
		err = rows.Scan(&k, &v)
		if err != nil {
			break
		}
		r = append(r, map[string]string{
			"k": k.String,
			"v": v.String,
		})
	}
	return r, err
}

func (m *mySQL) showVariables() ([]map[string]string, error) {
	sqlStr := "SHOW VARIABLES"
	return m.kvVal(sqlStr)
}

func (m *mySQL) getUserGrants(host, user string) (string, map[string]map[string]bool, []string, error) {
	r := map[string]map[string]bool{}
	var sortNumber []string
	var oldPass string
	sqlStr := "SHOW GRANTS FOR '" + com.AddSlashes(user) + "'@'" + com.AddSlashes(host) + "'"
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return oldPass, r, sortNumber, err
	}
	for rows.Next() {
		var v sql.NullString
		err = rows.Scan(&v)
		if err != nil {
			break
		}
		matchOn := reGrantOn.FindStringSubmatch(v.String)
		/*
			GRANT ALL PRIVILEGES ON *.* TO 'root'@'localhost' IDENTIFIED BY PASSWORD '*81F5E21E35407D884A6CD4A731AEBFB6AF209E1B' WITH GRANT OPTION
			matchOn :
			[
			  	"GRANT ALL PRIVILEGES ON *.* TO ",
			  	"ALL PRIVILEGES",
			  	"*.*"
			]
		*/
		if len(matchOn) > 0 {
			matchBrackets := reGrantBrackets.FindAllStringSubmatch(matchOn[1], -1)
			/*
				GRANT ALL PRIVILEGES ON *.* TO 'root'@'localhost' IDENTIFIED BY PASSWORD '*81F5E21E35407D884A6CD4A731AEBFB6AF209E1B' WITH GRANT OPTION
				matchBrackets :
				[
				  [
				    "ALL PRIVILEGES",
				    "ALL PRIVILEGES",
				    "",
				    ""
				  ]
				]
			*/
			if len(matchBrackets) > 0 {
				for _, val := range matchBrackets {
					if val[1] != `USAGE` {
						k := matchOn[2] + val[2]
						if _, ok := r[k]; !ok {
							r[k] = map[string]bool{}
							sortNumber = append(sortNumber, k)
						}
						r[k][val[1]] = true
					}
					if reGrantOption.MatchString(v.String) {
						k := matchOn[2] + val[2]
						if _, ok := r[k]; !ok {
							r[k] = map[string]bool{}
							sortNumber = append(sortNumber, k)
						}
						r[k]["GRANT OPTION"] = true
					}
				}
			}
		}
		matchIdent := reGrantIdent.FindStringSubmatch(v.String)
		if len(matchIdent) > 0 {
			oldPass = matchIdent[1]
		}
	}
	sqlStr = "SELECT SUBSTRING_INDEX(CURRENT_USER, '@', -1)"
	row, err := m.newParam().SetCollection(sqlStr).QueryRow()
	if err == nil {
		var v sql.NullString
		err = row.Scan(&v)
		if err != nil {
			return oldPass, r, sortNumber, err
		}
		m.Request().Form().Set(`host`, v.String)
	}
	var key string
	if len(m.dbName) == 0 || (r != nil && len(r) > 0) {
	} else {
		key = com.AddCSlashes(m.dbName, '%', '_', '\\')
	}
	key += ".*"
	r[key] = map[string]bool{}
	sortNumber = append(sortNumber, key)
	return oldPass, r, sortNumber, err
}

func (m *mySQL) listPrivileges() (bool, []map[string]string, error) {
	sqlStr := "SELECT User, Host FROM mysql."
	if len(m.dbName) == 0 {
		sqlStr += `user`
	} else {
		sqlStr += "db WHERE `" + strings.Replace(m.dbName, "`", "", -1) + "` LIKE Db"
	}
	sqlStr += " ORDER BY Host, User"
	res, err := m.kvVal(sqlStr)
	sysUser := true
	if err != nil || res == nil || len(res) == 0 {
		sysUser = false
		sqlStr = `SELECT SUBSTRING_INDEX(CURRENT_USER, '@', 1) AS User, SUBSTRING_INDEX(CURRENT_USER, '@', -1) AS Host`
		res, err = m.kvVal(sqlStr)
	}
	return sysUser, res, err
}

func (m *mySQL) showPrivileges() (*Privileges, error) {
	r := NewPrivileges()
	sqlStr := "SHOW PRIVILEGES"
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return r, err
	}
	for rows.Next() {
		v := &Privilege{}
		err = rows.Scan(&v.Privilege, &v.Context, &v.Comment)
		if err != nil {
			break
		}
		r.Privileges = append(r.Privileges, v)
	}
	return r, err
}

func (m *mySQL) processList() ([]*ProcessList, error) {
	r := []*ProcessList{}
	sqlStr := "SHOW FULL PROCESSLIST"
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return r, err
	}
	for rows.Next() {
		v := &ProcessList{}
		err = rows.Scan(&v.Id, &v.User, &v.Host, &v.Db, &v.Command, &v.Time, &v.State, &v.Info, &v.Progress)
		if err != nil {
			break
		}
		r = append(r, v)
	}
	return r, err
}

func (m *mySQL) showStatus() ([]map[string]string, error) {
	sqlStr := "SHOW STATUS"
	return m.kvVal(sqlStr)
}

func (m *mySQL) createDatabase(dbName, collate string) *Result {
	r := &Result{}
	r.SQL = "CREATE DATABASE `" + strings.Replace(dbName, "`", "", -1) + "`"
	if len(collate) > 0 {
		r.SQL += " COLLATE '" + com.AddSlashes(collate) + "'"
	}
	return r.Exec(m.newParam())
}

func (m *mySQL) dropDatabase(dbName string) *Result {
	r := &Result{}
	r.SQL = "DROP DATABASE `" + strings.Replace(dbName, "`", "", -1) + "`"
	return r.Exec(m.newParam())
}

func (m *mySQL) renameDatabase(newName, collate string) []*Result {
	newName = strings.Replace(newName, "`", "", -1)
	rs := []*Result{}
	r := m.createDatabase(newName, collate)
	rs = append(rs, r)
	if r.Error != nil {
		return rs
	}
	rGetTables := &Result{}
	rGetTables.start()
	tables, err := m.getTables()
	rGetTables.end()
	rGetTables.SQL = `SHOW TABLES`
	rGetTables.Error = err
	rs = append(rs, rGetTables)
	if err != nil {
		return rs
	}
	var sql string
	for key, table := range tables {
		table = com.AddCSlashes(table, '`')
		if key > 0 {
			sql += ", "
		}
		sql += "`" + table + "` TO `" + newName + "`.`" + table + "`"
	}
	if len(sql) > 0 {
		rRename := &Result{}
		rRename.SQL = "RENAME TABLE " + sql
		rRename = rRename.Exec(m.newParam())
		err = rRename.Error
		rs = append(rs, rRename)
	}
	if err == nil {
		rDrop := m.dropDatabase(m.dbName)
		rs = append(rs, rDrop)
	}
	return rs
}

func (m *mySQL) setLastSQL(sqlStr string) {
	m.Session().AddFlash(sqlStr, `lastSQL`)
}

func (m *mySQL) lastSQL() interface{} {
	return m.Flash(`lastSQL`)
}

// 获取数据库列表
func (m *mySQL) getDatabases() ([]string, error) {
	sqlStr := `SELECT SCHEMA_NAME FROM information_schema.SCHEMATA`
	if com.VersionCompare(m.getVersion(), `5`) < 0 {
		sqlStr = `SHOW DATABASES`
	}
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for rows.Next() {
		var v sql.NullString
		err := rows.Scan(&v)
		if err != nil {
			return nil, err
		}
		ret = append(ret, v.String)
	}
	return ret, nil
}

// 获取数据表列表
func (m *mySQL) getTables() ([]string, error) {
	sqlStr := `SHOW TABLES`
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return nil, err
	}
	ret := []string{}
	for rows.Next() {
		var v sql.NullString
		err := rows.Scan(&v)
		if err != nil {
			return nil, err
		}
		ret = append(ret, v.String)
	}
	return ret, nil
}

// 获取支持的字符集
func (m *mySQL) getCollations() (*Collations, error) {
	sqlStr := `SHOW COLLATION`
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return nil, err
	}
	ret := NewCollations()
	for rows.Next() {
		var v Collation
		err := rows.Scan(&v.Collation, &v.Charset, &v.Id, &v.Default, &v.Compiled, &v.Sortlen)
		if err != nil {
			return nil, err
		}
		coll, ok := ret.Collations[v.Charset.String]
		if !ok {
			coll = []Collation{}
		}
		if v.Default.Valid && len(v.Default.String) > 0 {
			ret.Defaults[v.Charset.String] = len(coll)
		}
		coll = append(coll, v)
		ret.Collations[v.Charset.String] = coll
	}
	return ret, nil
}

func (m *mySQL) getCollation(dbName string, collations *Collations) (string, error) {
	var err error
	if collations == nil {
		collations, err = m.getCollations()
		if err != nil {
			return ``, err
		}
	}
	sqlStr := "SHOW CREATE DATABASE `" + strings.Replace(dbName, "`", "", -1) + "`"
	row, err := m.newParam().SetCollection(sqlStr).QueryRow()
	if err != nil {
		return ``, err
	}
	var database sql.NullString
	var createDb sql.NullString
	err = row.Scan(&database, &createDb)
	if err != nil {
		return ``, err
	}
	matches := reCollate.FindStringSubmatch(createDb.String)
	if len(matches) > 1 {
		return matches[1], nil
	}
	matches = reCharacter.FindStringSubmatch(createDb.String)
	if len(matches) > 1 {
		if idx, ok := collations.Defaults[matches[1]]; ok {
			return collations.Collations[matches[1]][idx].Collation.String, nil
		}
	}

	return ``, nil
}

func (m *mySQL) getTableStatus(dbName string, tableName string, fast bool) (map[string]*TableStatus, error) {
	sqlStr := `SHOW TABLE STATUS`
	if len(dbName) > 0 {
		sqlStr += " FROM `" + strings.Replace(dbName, "`", "", -1) + "`"
	}
	if len(tableName) > 0 {
		tableName = com.AddSlashes(tableName, '_', '%')
		tableName = `'` + tableName + `'`
		sqlStr += ` LIKE ` + tableName
	}
	rows, err := m.newParam().SetCollection(sqlStr).Query()
	if err != nil {
		return nil, err
	}
	ret := map[string]*TableStatus{}
	for rows.Next() {
		v := &TableStatus{}
		err := rows.Scan(&v.Name, &v.Engine, &v.Version, &v.Row_format, &v.Rows, &v.Avg_row_length, &v.Data_length, &v.Max_data_length, &v.Index_length, &v.Data_free, &v.Auto_increment, &v.Create_time, &v.Update_time, &v.Check_time, &v.Collation, &v.Checksum, &v.Create_options, &v.Comment)
		if err != nil {
			return nil, err
		}
		if v.Engine.String == `InnoDB` {
			v.Comment.String = reInnoDBComment.ReplaceAllString(v.Comment.String, `$1`)
		}
		ret[v.Name.String] = v
		if len(tableName) > 0 {
			return ret, nil
		}
	}
	return ret, nil
}

func (m *mySQL) newParam() *factory.Param {
	return factory.NewParam(m.db)
}

func (m *mySQL) getVersion() string {
	if len(m.version) > 0 {
		return m.version
	}
	row, err := m.newParam().SetCollection(`SELECT version()`).QueryRow()
	if err != nil {
		return err.Error()
	}
	var v sql.NullString
	err = row.Scan(&v)
	if err != nil {
		return err.Error()
	}
	m.version = v.String
	return v.String
}

func (m *mySQL) baseInfo() error {
	dbList, err := m.getDatabases()
	if err != nil {
		return err
	}
	m.Set(`dbList`, dbList)
	if len(m.DbAuth.Db) > 0 {
		tableList, err := m.getTables()
		if err != nil {
			return err
		}
		m.Set(`tableList`, tableList)
	}

	m.Set(`dbVersion`, m.getVersion())
	return nil
}
