package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// protoField represents a single field in a proto message.
type protoField struct {
	repeated    bool
	typeName    string
	name        string
	isOneof     bool
	oneofName   string
	oneofFields []oneofOption
	isMap       bool
}

type oneofOption struct {
	typeName string
	name     string
}

type protoMessage struct {
	name   string
	fields []protoField
}

var (
	// stringAliases tracks `type X string` declarations which map to proto string.
	stringAliases = map[string]bool{}
	// knownStructs tracks struct type names from the package.
	knownStructs = map[string]bool{}
)

func main() {
	root, err := findProjectRoot()
	if err != nil {
		log.Fatal(err)
	}

	jsDir := filepath.Join(root, "jsonSchema")
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, jsDir, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("Failed to parse jsonSchema package: %v", err)
	}

	pkg, ok := pkgs["jsonSchema"]
	if !ok {
		log.Fatal("jsonSchema package not found")
	}

	// First pass: collect type aliases and struct names.
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				switch underlying := ts.Type.(type) {
				case *ast.StructType:
					_ = underlying
					knownStructs[ts.Name.Name] = true
				case *ast.Ident:
					if underlying.Name == "string" {
						stringAliases[ts.Name.Name] = true
					}
				}
			}
		}
	}

	// Second pass: collect struct definitions as proto messages.
	var messages []protoMessage
	seen := map[string]bool{}
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if seen[ts.Name.Name] {
					continue
				}
				seen[ts.Name.Name] = true
				msg := parseStruct(ts.Name.Name, st)
				messages = append(messages, msg)
			}
		}
	}

	// Sort: Definition first, then alphabetical.
	sort.Slice(messages, func(i, j int) bool {
		if messages[i].name == "Definition" {
			return true
		}
		if messages[j].name == "Definition" {
			return false
		}
		return messages[i].name < messages[j].name
	})

	// Generate proto file.
	var buf bytes.Buffer
	writeProtoHeader(&buf)
	for _, msg := range messages {
		writeMessage(&buf, msg)
	}
	writeInfrastructureMessages(&buf)
	writeService(&buf)

	protoPath := filepath.Join(root, "objectweaver.proto")
	if err := os.WriteFile(protoPath, buf.Bytes(), 0644); err != nil {
		log.Fatalf("Failed to write proto file: %v", err)
	}
	fmt.Println("Generated:", protoPath)

	compileProto(root)
	fmt.Println("Proto compilation complete")
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

func parseStruct(name string, st *ast.StructType) protoMessage {
	msg := protoMessage{name: name}
	for _, field := range st.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		fieldName := field.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue
		}
		pf := resolveFieldType(fieldName, field.Type)
		msg.fields = append(msg.fields, pf)
	}
	return msg
}

func resolveFieldType(fieldName string, expr ast.Expr) protoField {
	name := toProtoFieldName(fieldName)

	switch t := expr.(type) {
	case *ast.Ident:
		return resolveIdent(name, t)
	case *ast.StarExpr:
		pf := resolveFieldType(fieldName, t.X)
		return pf
	case *ast.ArrayType:
		return resolveArrayType(name, t)
	case *ast.MapType:
		return resolveMapType(name, t)
	case *ast.SelectorExpr:
		// e.g. pb.DetailedField
		return protoField{typeName: t.Sel.Name, name: name}
	case *ast.InterfaceType:
		return makeOneofField(name)
	default:
		return protoField{typeName: "string", name: name}
	}
}

func resolveIdent(name string, ident *ast.Ident) protoField {
	switch ident.Name {
	case "string":
		return protoField{typeName: "string", name: name}
	case "int", "int32":
		return protoField{typeName: "int32", name: name}
	case "int64":
		return protoField{typeName: "int64", name: name}
	case "float32":
		return protoField{typeName: "float", name: name}
	case "float64":
		return protoField{typeName: "double", name: name}
	case "bool":
		return protoField{typeName: "bool", name: name}
	case "byte":
		return protoField{typeName: "bytes", name: name}
	case "any":
		return makeOneofField(name)
	}
	if stringAliases[ident.Name] {
		return protoField{typeName: "string", name: name}
	}
	if knownStructs[ident.Name] {
		return protoField{typeName: ident.Name, name: name}
	}
	return protoField{typeName: "string", name: name}
}

func resolveArrayType(name string, arr *ast.ArrayType) protoField {
	// []byte → bytes
	if ident, ok := arr.Elt.(*ast.Ident); ok && ident.Name == "byte" {
		return protoField{typeName: "bytes", name: name}
	}
	// [][]byte → repeated bytes
	if inner, ok := arr.Elt.(*ast.ArrayType); ok {
		if ident, ok := inner.Elt.(*ast.Ident); ok && ident.Name == "byte" {
			return protoField{repeated: true, typeName: "bytes", name: name}
		}
	}
	// []T → repeated T
	inner := resolveFieldType("_", arr.Elt)
	return protoField{repeated: true, typeName: inner.typeName, name: name}
}

func resolveMapType(name string, m *ast.MapType) protoField {
	// Check if value is interface{}/any → google.protobuf.Struct
	if isInterfaceOrAny(m.Value) {
		return protoField{typeName: "google.protobuf.Struct", name: name}
	}
	keyType := goTypeToProtoScalar(m.Key)
	valType := goTypeToProtoScalar(m.Value)
	return protoField{
		typeName: fmt.Sprintf("map<%s, %s>", keyType, valType),
		name:     name,
		isMap:    true,
	}
}

func isInterfaceOrAny(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.InterfaceType:
		return true
	case *ast.Ident:
		return t.Name == "any"
	}
	return false
}

func goTypeToProtoScalar(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "string":
			return "string"
		case "int", "int32":
			return "int32"
		case "int64":
			return "int64"
		case "float32":
			return "float"
		case "float64":
			return "double"
		case "bool":
			return "bool"
		}
		if stringAliases[t.Name] {
			return "string"
		}
		if knownStructs[t.Name] {
			return t.Name
		}
		return "string"
	case *ast.StarExpr:
		return goTypeToProtoScalar(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return "string"
}

func makeOneofField(name string) protoField {
	return protoField{
		isOneof:   true,
		oneofName: name,
		name:      name,
		oneofFields: []oneofOption{
			{typeName: "double", name: "number_value"},
			{typeName: "string", name: "string_value"},
			{typeName: "bool", name: "bool_value"},
		},
	}
}

func toProtoFieldName(goFieldName string) string {
	if goFieldName == "" || goFieldName == "_" {
		return ""
	}
	// All-caps → all lower (URL → url, ID → id, N → n)
	if strings.ToUpper(goFieldName) == goFieldName {
		return strings.ToLower(goFieldName)
	}
	// Lowercase first character
	return strings.ToLower(goFieldName[:1]) + goFieldName[1:]
}

func writeProtoHeader(buf *bytes.Buffer) {
	buf.WriteString(`syntax = "proto3";

package jsonSchema;

option go_package = "./grpc";

import "google/protobuf/struct.proto";

`)
}

func writeMessage(buf *bytes.Buffer, msg protoMessage) {
	fmt.Fprintf(buf, "// %s message\nmessage %s {\n", msg.name, msg.name)
	fieldNum := 1
	for _, f := range msg.fields {
		if f.isOneof {
			fmt.Fprintf(buf, "  oneof %s {\n", f.oneofName)
			for _, opt := range f.oneofFields {
				fmt.Fprintf(buf, "    %s %s = %d;\n", opt.typeName, opt.name, fieldNum)
				fieldNum++
			}
			buf.WriteString("  }\n")
		} else {
			prefix := ""
			if f.repeated {
				prefix = "repeated "
			}
			fmt.Fprintf(buf, "  %s%s %s = %d;\n", prefix, f.typeName, f.name, fieldNum)
			fieldNum++
		}
	}
	buf.WriteString("}\n\n")
}

func writeInfrastructureMessages(buf *bytes.Buffer) {
	buf.WriteString(`// Choice message for epistemic validation results
message Choice {
  int32 score = 1;
  double confidence = 2;
  google.protobuf.Struct value = 3;
  repeated double embedding = 4;
}

// FieldMetadata contains metadata for a single field
message FieldMetadata {
  int32 tokensUsed = 1;
  double cost = 2;
  string modelUsed = 3;
  repeated Choice choices = 4;
}

// DetailedField contains both the value and metadata for a field
message DetailedField {
  google.protobuf.Struct value = 1;
  FieldMetadata metadata = 2;
}

// StreamingResponse message for the stream method
message StreamingResponse {
  google.protobuf.Struct data = 1;
  double usdCost = 2;
  string status = 3;
  map<string, DetailedField> detailedData = 4;
}

`)
}

func writeService(buf *bytes.Buffer) {
	buf.WriteString(`// The JSONSchemaService defines a service for generating JSON objects based on a schema definition.
service JSONSchemaService {
  // Standard request-response RPC
  rpc GenerateObject(RequestBody) returns (Response);

  // Server-side streaming RPC
  rpc StreamGeneratedObjects(RequestBody) returns (stream StreamingResponse);
}
`)
}

func compileProto(root string) {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		home, _ := os.UserHomeDir()
		goPath = filepath.Join(home, "go")
	}

	path := os.Getenv("PATH")
	os.Setenv("PATH", filepath.Join(goPath, "bin")+":"+path)

	cmd := exec.Command("protoc",
		"--go_out=.",
		"--go-grpc_out=.",
		"objectweaver.proto",
	)
	cmd.Dir = root
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to compile proto: %v", err)
	}
}
