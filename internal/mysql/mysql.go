package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/takaaa220/db2jsonschema/internal"
)

type ConnectionSetting struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type mysqlDialect struct {
	connectionSetting ConnectionSetting
}

func NewMysqlDialect(connectionSetting ConnectionSetting) *mysqlDialect {
	return &mysqlDialect{connectionSetting: connectionSetting}
}

func (d *mysqlDialect) GetTables() ([]internal.Table, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", d.connectionSetting.User, d.connectionSetting.Password, d.connectionSetting.Host, d.connectionSetting.Port, d.connectionSetting.Database))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	return getDBSchema(db, d.connectionSetting.Database)
}

func getDBSchema(db *sql.DB, database string) ([]internal.Table, error) {
	query := fmt.Sprintf(`
	SELECT
		TABLE_NAME,
		COLUMN_NAME,
		DATA_TYPE,
		COLUMN_TYPE,
		IS_NULLABLE,
		CHARACTER_MAXIMUM_LENGTH,
		COLUMN_DEFAULT
	FROM INFORMATION_SCHEMA.COLUMNS
	WHERE TABLE_SCHEMA = "%s"`, database)

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}

	tables := map[string]internal.Table{}

	for rows.Next() {
		var tableName, columnName, dataType, columnType, isNullable string
		var columnDefault *string
		var maxLength sql.NullInt64
		err = rows.Scan(&tableName, &columnName, &dataType, &columnType, &isNullable, &maxLength, &columnDefault)
		if err != nil {
			return nil, err
		}

		table, found := tables[tableName]
		if !found {
			table = internal.Table{Name: tableName}
		}

		column := internal.Column{
			Name:     columnName,
			Nullable: isNullable == "YES",
			Default:  columnDefault,
		}

		t, err := NewColumnType(columnType)
		if err != nil {
			return nil, err
		}
		column.Type = t

		if maxLength.Valid {
			column.MaxLength = int(maxLength.Int64)
		}

		if strings.Contains(columnType, "unsigned") {
			column.Unsigned = true
		}

		if column.Type == "enum" {
			s := strings.Split(
				strings.TrimRight(strings.TrimLeft(columnType, "enum("), ")"),
				",",
			)
			for i := range s {
				s[i] = strings.Trim(s[i], "'")
			}

			column.Enum = s
		}

		table.Columns = append(table.Columns, column)

		tables[tableName] = table
	}

	res := make([]internal.Table, 0, len(tables))
	for _, t := range tables {
		res = append(res, t)
	}

	return res, nil
}

func NewColumnType(columnType string) (internal.ColumnType, error) {
	if strings.Contains(columnType, "int") {
		return internal.ColumnTypeInteger, nil
	}

	if strings.Contains(columnType, "float") || strings.Contains(columnType, "double") || strings.Contains(columnType, "decimal") {
		return internal.ColumnTypeFloat, nil
	}

	if columnType == "tinyint(1)" {
		return internal.ColumnTypeBoolean, nil
	}

	if strings.Contains(columnType, "char") || strings.Contains(columnType, "text") {
		return internal.ColumnTypeString, nil
	}

	if strings.Contains(columnType, "enum") {
		return internal.ColumnTypeEnum, nil
	}

	if strings.Contains(columnType, "datetime") {
		return internal.ColumnTypeDatetime, nil
	}

	if strings.Contains(columnType, "date") {
		return internal.ColumnTypeDate, nil
	}

	if strings.Contains(columnType, "timestamp") {
		return internal.ColumnTypeDatetime, nil
	}

	if strings.Contains(columnType, "json") {
		return internal.ColumnTypeJSON, nil
	}

	return "", fmt.Errorf("unsupported type: %s", columnType)
}
