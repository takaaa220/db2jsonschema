package internal

import (
	"encoding/json"
	"fmt"
)

type GenSetting struct {
	DateTimePattern string
}

type generator struct {
	setting GenSetting
	dialect dialect
}

func NewGenerator(setting GenSetting, dialect dialect) *generator {
	return &generator{setting: setting, dialect: dialect}
}

func (g *generator) Gen() error {
	tables, err := g.dialect.GetTables()
	if err != nil {
		return err
	}

	jsonTableSchemas := []JSONSchemaTable{}
	for _, t := range tables {
		schema, err := g.genTableJSONSchema(t)
		if err != nil {
			return err
		}

		jsonTableSchemas = append(jsonTableSchemas, *schema)
	}

	allTablesSchema, err := g.genAllTablesJSONSchema(jsonTableSchemas)
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

type JSONSchemaTable struct {
	Schema     string                      `json:"$schema"`
	Type       JSONSchemaType              `json:"type"`
	Title      string                      `json:"title"`
	Properties map[string]JSONSchemaColumn `json:"properties"`
	Required   []string                    `json:"required"`
}

type JSONSchemaColumn struct {
	Type        JSONSchemaType `json:"type"`
	MaxLength   int            `json:"maxLength,omitempty"`
	Enum        []string       `json:"enum,omitempty"`
	Format      string         `json:"format,omitempty"`
	Pattern     string         `json:"pattern,omitempty"`
	Description string         `json:"description,omitempty"`
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

func (g *generator) genTableJSONSchema(table Table) (*JSONSchemaTable, error) {
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
			column.Pattern = g.setting.DateTimePattern
			column.Description = "(datetime)"
		}

		if !dbColumn.Nullable && dbColumn.Default == nil {
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
	Schema     string                          `json:"$schema"`
	Type       JSONSchemaType                  `json:"type"`
	Properties map[string]TableArrayJSONSchema `json:"properties"`
}

type TableArrayJSONSchema struct {
	Type  string          `json:"type"`
	Items JSONSchemaTable `json:"items"`
}

func (g *generator) genAllTablesJSONSchema(tables []JSONSchemaTable) (AllTablesJSONSchema, error) {
	schema := AllTablesJSONSchema{
		Schema:     "http://json-schema.org/draft-07/schema#",
		Type:       "object",
		Properties: map[string]TableArrayJSONSchema{},
	}

	for _, t := range tables {
		schema.Properties[t.Title] = TableArrayJSONSchema{
			Type:  "array",
			Items: t,
		}
	}

	return schema, nil
}
