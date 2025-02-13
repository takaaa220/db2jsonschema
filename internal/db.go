package internal

import (
	"fmt"
	"strings"
)

type dialect interface {
	GetTables() ([]Table, error)
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
	Default   *string
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
