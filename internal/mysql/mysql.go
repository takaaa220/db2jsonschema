package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type ConnectionSetting struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func Gen(c ConnectionSetting) error {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.User, c.Password, c.Host, c.Port, c.Database))
	if err != nil {
		return err
	}
	defer db.Close()

	tables, err := getDBSchema(db, c.Database)
	if err != nil {
		return err
	}

	jsonTableSchemas := []JSONSchemaTable{}
	for _, t := range tables {
		schema, err := genTableJSONSchema(t)
		if err != nil {
			return err
		}

		jsonTableSchemas = append(jsonTableSchemas, *schema)
	}

	allTablesSchema, err := genAllTablesJSONSchema(jsonTableSchemas)
	if err != nil {
		return err
	}

	b, err := json.Marshal(allTablesSchema)
	if err != nil {
		return err
	}

	// TODO: output to file
	fmt.Println(string(b))

	return nil
}

func getDBSchema(db *sql.DB, database string) ([]Table, error) {
	query := fmt.Sprintf(`
	SELECT
		TABLE_NAME,
		COLUMN_NAME,
		DATA_TYPE,
		COLUMN_TYPE,
		IS_NULLABLE,
		CHARACTER_MAXIMUM_LENGTH
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

	tables := map[string]Table{}

	for rows.Next() {
		var tableName, columnName, dataType, columnType, isNullable string
		var maxLength sql.NullInt64
		err = rows.Scan(&tableName, &columnName, &dataType, &columnType, &isNullable, &maxLength)
		if err != nil {
			return nil, err
		}

		table, found := tables[tableName]
		if !found {
			table = Table{Name: tableName}
		}

		column := Column{
			Name:     columnName,
			Nullable: isNullable == "YES",
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

	res := make([]Table, 0, len(tables))
	for _, t := range tables {
		res = append(res, t)
	}

	return res, nil
}

type Table struct {
	Name    string
	Columns []Column
}

func (t Table) String() string {
	columns := []string{}
	for _, c := range t.Columns {
		columns = append(columns, c.String())
	}

	return fmt.Sprintf("Table: %s\n  %s", t.Name, strings.Join(columns, "\n  "))
}

type Column struct {
	Name      string
	Type      ColumnType
	Nullable  bool
	MaxLength int
	Enum      []string
	Unsigned  bool
}

func (c Column) String() string {
	return fmt.Sprintf("Name: %s, Type: %s, Nullable: %v, MaxLength: %d, Enum: %v, Unsigned: %v", c.Name, c.Type, c.Nullable, c.MaxLength, c.Enum, c.Unsigned)
}

type ColumnType string

const (
	ColumnTypeJSON     ColumnType = "json"
	ColumnTypeInteger  ColumnType = "integer"
	ColumnTypeFloat    ColumnType = "float"
	ColumnTypeBoolean  ColumnType = "boolean"
	ColumnTypeString   ColumnType = "string"
	ColumnTypeDate     ColumnType = "date"
	ColumnTypeDatetime ColumnType = "datetime"
	ColumnTypeEnum     ColumnType = "enum"
)

func NewColumnType(columnType string) (ColumnType, error) {
	if strings.Contains(columnType, "int") {
		return ColumnTypeInteger, nil
	}

	if strings.Contains(columnType, "float") || strings.Contains(columnType, "double") || strings.Contains(columnType, "decimal") {
		return ColumnTypeFloat, nil
	}

	if columnType == "tinyint(1)" {
		return ColumnTypeBoolean, nil
	}

	if strings.Contains(columnType, "char") || strings.Contains(columnType, "text") {
		return ColumnTypeString, nil
	}

	if strings.Contains(columnType, "enum") {
		return ColumnTypeEnum, nil
	}

	if strings.Contains(columnType, "datetime") {
		return ColumnTypeDatetime, nil
	}

	if strings.Contains(columnType, "date") {
		return ColumnTypeDate, nil
	}

	if strings.Contains(columnType, "timestamp") {
		return ColumnTypeDatetime, nil
	}

	if strings.Contains(columnType, "json") {
		return ColumnTypeJSON, nil
	}

	return "", fmt.Errorf("unsupported type: %s", columnType)
}

type JSONSchemaTable struct {
	Schema     string                      `json:"$schema"`
	Type       JSONSchemaType              `json:"type"`
	Title      string                      `json:"title"`
	Properties map[string]JSONSchemaColumn `json:"properties"`
	Required   []string                    `json:"required"`
}

type JSONSchemaColumn struct {
	Type      JSONSchemaType `json:"type"`
	MaxLength int            `json:"maxLength,omitempty"`
	Enum      []string       `json:"enum,omitempty"`
	Format    string         `json:"format,omitempty"`
}

type JSONSchemaType string

const (
	JSONSchemaTypeString  JSONSchemaType = "string"
	JSONSchemaTypeInteger JSONSchemaType = "integer"
	JSONSchemaTypeNumber  JSONSchemaType = "number"
	JSONSchemaTypeBoolean JSONSchemaType = "boolean"
	JSONSchemaTypeObject  JSONSchemaType = "object"
	JSONSchemaTypeArray   JSONSchemaType = "array"
	JSONSchemaTypeNull    JSONSchemaType = "null"
)

func genTableJSONSchema(table Table) (*JSONSchemaTable, error) {
	schema := JSONSchemaTable{
		Schema:     "http://json-schema.org/draft-07/schema#",
		Type:       "object",
		Title:      table.Name,
		Properties: map[string]JSONSchemaColumn{},
		Required:   []string{},
	}

	for _, dbColumn := range table.Columns {
		t, err := convertIntoJSONSchemaType(dbColumn.Type)
		if err != nil {
			return nil, err
		}

		column := JSONSchemaColumn{
			Type:      t,
			MaxLength: dbColumn.MaxLength,
			Enum:      dbColumn.Enum,
		}

		if len(dbColumn.Enum) != 0 {
			column.Enum = dbColumn.Enum
		}

		if dbColumn.MaxLength > 0 {
			column.MaxLength = dbColumn.MaxLength
		}

		if dbColumn.Type == ColumnTypeDate {
			column.Format = "date"
		}

		if dbColumn.Type == ColumnTypeDatetime {
			column.Format = "date-time"
		}

		if !dbColumn.Nullable {
			schema.Required = append(schema.Required, dbColumn.Name)
		}

		schema.Properties[dbColumn.Name] = column
	}

	return &schema, nil
}

func convertIntoJSONSchemaType(t ColumnType) (JSONSchemaType, error) {
	switch t {
	case ColumnTypeInteger:
		return "integer", nil
	case ColumnTypeFloat:
		return "number", nil
	case ColumnTypeBoolean:
		return "boolean", nil
	case ColumnTypeString, ColumnTypeEnum, ColumnTypeDate, ColumnTypeDatetime, ColumnTypeJSON:
		return "string", nil
	default:
		return "", fmt.Errorf("unsupported type: %s", t)
	}
}

type AllTablesJSONSchema struct {
	Schema     string                     `json:"$schema"`
	Type       JSONSchemaType             `json:"type"`
	Properties map[string]JSONSchemaTable `json:"properties"`
}

func genAllTablesJSONSchema(tables []JSONSchemaTable) (AllTablesJSONSchema, error) {
	schema := AllTablesJSONSchema{
		Schema:     "http://json-schema.org/draft-07/schema#",
		Type:       "object",
		Properties: map[string]JSONSchemaTable{},
	}

	for _, t := range tables {
		schema.Properties[t.Title] = t
	}

	return schema, nil
}
