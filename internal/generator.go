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

func (g *generator) Gen() ([]byte, error) {
	tables, err := g.dialect.GetTables()
	if err != nil {
		return nil, err
	}

	jsonTableSchemas := []JSONSchemaObject{}
	for _, t := range tables {
		schema, err := g.genTableJSONSchema(t)
		if err != nil {
			return nil, err
		}

		jsonTableSchemas = append(jsonTableSchemas, *schema)
	}

	allTablesSchema, err := g.genAllTablesJSONSchema(jsonTableSchemas)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(allTablesSchema)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type JSONSchemaObject struct {
	Schema      string                      `json:"$schema,omitempty"`
	Type        JSONSchemaType              `json:"type,omitempty"`
	Title       string                      `json:"title,omitempty"`
	Description string                      `json:"description,omitempty"`
	Properties  map[string]JSONSchemaObject `json:"properties,omitempty"`
	Definitions map[string]JSONSchemaObject `json:"definitions,omitempty"`
	Required    []string                    `json:"required,omitempty"`
	MaxLength   int                         `json:"maxLength,omitempty"`
	Enum        []string                    `json:"enum,omitempty"`
	Format      string                      `json:"format,omitempty"`
	Pattern     string                      `json:"pattern,omitempty"`
	Items       *JSONSchemaObject           `json:"items,omitempty"`
	Ref         string                      `json:"$ref,omitempty"`
	AnyOf       []JSONSchemaObject          `json:"anyOf,omitempty"`
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

const (
	TestFixturesRaw = "testfixtures-raw"
)

func (g *generator) genTableJSONSchema(table Table) (*JSONSchemaObject, error) {
	schema := JSONSchemaObject{
		Type:       JSONSchemaTypeObject,
		Title:      table.Name,
		Properties: map[string]JSONSchemaObject{},
		Required:   []string{},
	}

	for _, dbColumn := range table.Columns {
		t, err := convertIntoJSONSchemaType(dbColumn.Type)
		if err != nil {
			return nil, err
		}

		column := JSONSchemaObject{
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

		schema.Properties[dbColumn.Name] = JSONSchemaObject{
			AnyOf: []JSONSchemaObject{
				column,
				{Ref: "#/definitions/" + TestFixturesRaw},
			},
		}
	}

	return &schema, nil
}

func convertIntoJSONSchemaType(t ColumnType) (JSONSchemaType, error) {
	switch t {
	case ColumnTypeInteger:
		return JSONSchemaTypeInteger, nil
	case ColumnTypeFloat:
		return JSONSchemaTypeNumber, nil
	case ColumnTypeBoolean:
		return JSONSchemaTypeBoolean, nil
	case ColumnTypeString, ColumnTypeEnum, ColumnTypeDate, ColumnTypeDatetime, ColumnTypeJSON:
		return JSONSchemaTypeString, nil
	default:
		return "", fmt.Errorf("unsupported type: %s", t)
	}
}

func (g *generator) genAllTablesJSONSchema(tables []JSONSchemaObject) (JSONSchemaObject, error) {
	schema := JSONSchemaObject{
		Schema:     "http://json-schema.org/draft-07/schema#",
		Type:       JSONSchemaTypeObject,
		Properties: map[string]JSONSchemaObject{},
		Definitions: map[string]JSONSchemaObject{
			TestFixturesRaw: {
				Type:    JSONSchemaTypeString,
				Pattern: "RAW=.*",
			},
		},
	}

	for _, t := range tables {
		t := t
		schema.Properties[t.Title] = JSONSchemaObject{
			Type: JSONSchemaTypeArray,
			Items: &JSONSchemaObject{
				Ref: fmt.Sprintf("#/definitions/%s", t.Title),
			},
		}
		schema.Definitions[t.Title] = t
	}

	return schema, nil
}
