package core

import (
	"testing"
)

func TestSchemaFromBasic(t *testing.T) {
	type Request struct {
		Name string `json:"name" description:"The person's name"`
		Age  int    `json:"age" description:"Age in years"`
	}

	schema := SchemaFrom[Request]()

	if schema.Type != "object" {
		t.Fatalf("expected type object, got %s", schema.Type)
	}
	if len(schema.Required) != 2 {
		t.Fatalf("expected 2 required fields, got %d", len(schema.Required))
	}

	nameProp, ok := schema.Properties["name"]
	if !ok {
		t.Fatal("missing property 'name'")
	}
	if nameProp.Type != "string" {
		t.Errorf("expected name type string, got %s", nameProp.Type)
	}
	if nameProp.Description != "The person's name" {
		t.Errorf("expected description, got %s", nameProp.Description)
	}

	ageProp, ok := schema.Properties["age"]
	if !ok {
		t.Fatal("missing property 'age'")
	}
	if ageProp.Type != "integer" {
		t.Errorf("expected age type integer, got %s", ageProp.Type)
	}
}

func TestSchemaFromOptional(t *testing.T) {
	type Config struct {
		Host    string `json:"host"`
		Port    int    `json:"port,omitempty"`
		Verbose bool   `json:"verbose,omitempty"`
	}

	schema := SchemaFrom[Config]()

	if len(schema.Required) != 1 || schema.Required[0] != "host" {
		t.Errorf("expected required=[host], got %v", schema.Required)
	}
}

func TestSchemaFromNested(t *testing.T) {
	type Address struct {
		Street string `json:"street"`
		City   string `json:"city"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	schema := SchemaFrom[Person]()

	addrProp, ok := schema.Properties["address"]
	if !ok {
		t.Fatal("missing property 'address'")
	}
	if addrProp.Type != "object" {
		t.Errorf("expected address type object, got %s", addrProp.Type)
	}
	if len(addrProp.Properties) != 2 {
		t.Errorf("expected 2 nested properties, got %d", len(addrProp.Properties))
	}
	if len(addrProp.Required) != 2 {
		t.Errorf("expected 2 required nested fields, got %d", len(addrProp.Required))
	}
}

func TestSchemaFromSlice(t *testing.T) {
	type TaggedItem struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	schema := SchemaFrom[TaggedItem]()

	tagsProp, ok := schema.Properties["tags"]
	if !ok {
		t.Fatal("missing property 'tags'")
	}
	if tagsProp.Type != "array" {
		t.Errorf("expected tags type array, got %s", tagsProp.Type)
	}
	if tagsProp.Items == nil || tagsProp.Items.Type != "string" {
		t.Error("expected items type string")
	}
}

func TestSchemaFromPointer(t *testing.T) {
	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Inner *Inner `json:"inner,omitempty"`
		Count *int   `json:"count,omitempty"`
	}

	schema := SchemaFrom[Outer]()

	if len(schema.Required) != 0 {
		t.Errorf("expected no required fields, got %v", schema.Required)
	}

	innerProp, ok := schema.Properties["inner"]
	if !ok {
		t.Fatal("missing property 'inner'")
	}
	if innerProp.Type != "object" {
		t.Errorf("expected inner type object, got %s", innerProp.Type)
	}

	countProp, ok := schema.Properties["count"]
	if !ok {
		t.Fatal("missing property 'count'")
	}
	if countProp.Type != "integer" {
		t.Errorf("expected count type integer, got %s", countProp.Type)
	}
}

func TestSchemaFromEnum(t *testing.T) {
	type Priority struct {
		Level string `json:"level" enum:"low,medium,high" description:"Priority level"`
	}

	schema := SchemaFrom[Priority]()

	levelProp := schema.Properties["level"]
	if len(levelProp.Enum) != 3 {
		t.Errorf("expected 3 enum values, got %d", len(levelProp.Enum))
	}
	if levelProp.Enum[0] != "low" || levelProp.Enum[1] != "medium" || levelProp.Enum[2] != "high" {
		t.Errorf("unexpected enum values: %v", levelProp.Enum)
	}
}

func TestSchemaFromSkipsJsonDash(t *testing.T) {
	type Hidden struct {
		Public  string `json:"public"`
		Private string `json:"-"`
	}

	schema := SchemaFrom[Hidden]()

	if len(schema.Properties) != 1 {
		t.Errorf("expected 1 property, got %d", len(schema.Properties))
	}
	if _, ok := schema.Properties["private"]; ok {
		t.Error("json:\"-\" field should be skipped")
	}
}

func TestSchemaFromFloatAndBool(t *testing.T) {
	type Numeric struct {
		Score   float64 `json:"score"`
		Enabled bool    `json:"enabled"`
	}

	schema := SchemaFrom[Numeric]()

	if schema.Properties["score"].Type != "number" {
		t.Errorf("expected score type number, got %s", schema.Properties["score"].Type)
	}
	if schema.Properties["enabled"].Type != "boolean" {
		t.Errorf("expected enabled type boolean, got %s", schema.Properties["enabled"].Type)
	}
}

func TestSchemaFromNestedSliceOfStructs(t *testing.T) {
	type Item struct {
		Name string `json:"name"`
	}
	type List struct {
		Items []Item `json:"items"`
	}

	schema := SchemaFrom[List]()

	itemsProp := schema.Properties["items"]
	if itemsProp.Type != "array" {
		t.Fatalf("expected array, got %s", itemsProp.Type)
	}
	if itemsProp.Items == nil {
		t.Fatal("expected items schema")
	}
	if itemsProp.Items.Type != "object" {
		t.Errorf("expected items element type object, got %s", itemsProp.Items.Type)
	}
	if len(itemsProp.Items.Properties) != 1 {
		t.Errorf("expected 1 nested property, got %d", len(itemsProp.Items.Properties))
	}
}
