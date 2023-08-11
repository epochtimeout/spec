// --- Declarations ---
%{
package parser

import (
	"fmt"

	"github.com/basecomplextech/spec/lang/ast"
)
%}

// union defines yySymType body.
%union {
	// Tokens
	ident   string
	integer int
	string  string

	// Import
	import_ *ast.Import
	imports []*ast.Import

	// Option
	option  *ast.Option
	options []*ast.Option

	// Definition
	definition  *ast.Definition
	definitions []*ast.Definition

	// Enum
	enum_value  *ast.EnumValue
	enum_values []*ast.EnumValue

	// Message
	message_field  *ast.MessageField
	message_fields []*ast.MessageField

	// Struct
	struct_field  *ast.StructField
	struct_fields []*ast.StructField

	// Type
	type_ *ast.Type
}

// keywords
%token ENUM
%token IMPORT
%token MESSAGE
%token OPTIONS
%token STRUCT

// general
%token <ident>   IDENT
%token <integer> INTEGER
%token <string>  STRING
%token <ident>   MESSAGE
%type  <ident>   keyword

// import
%type <import_> import
%type <imports> import_list
%type <imports> imports

// option
%type <option>  option
%type <options> option_list
%type <options> options

// definitions
%type <definition>  definition
%type <definitions> definitions

// enum
%type <definition>  enum
%type <enum_value>  enum_value
%type <enum_values> enum_values

// message
%type <definition>      message
%type <message_field>   message_field
%type <message_fields> 	message_fields
%type <ident>           message_field_name

// struct
%type <definition>      struct
%type <struct_field>    struct_field
%type <struct_fields>   struct_fields
%type <ident>           struct_field_name

// type
%type <type_> type
%type <type_> base_type

// start
%start file

%%

keyword:
	IMPORT
    {
        $$ = "import"
    }
	| MESSAGE
    {
        $$ = "message"
    }
	| OPTIONS
    {
        $$ = "options"
    }
	| STRUCT
    {
        $$ = "struct"
    }
	;

// file

file: imports options definitions
	{ 
		file := &ast.File{
			Imports:     $1,
			Options:     $2,
			Definitions: $3,
		}
		setLexerResult(yylex, file)
	}

// import

import:
	STRING
	{ 
		if debugParser {
			fmt.Println("import ", $1)
		}
		$$ = &ast.Import{
			ID: trimString($1),
		}
	}
	| IDENT STRING
	{
		if debugParser {
			fmt.Println("import ", $1, $2)
		}
		$$ = &ast.Import{
			Alias: $1,
			ID:    trimString($2),
		}
	}

import_list:
	// Empty
	{ 
		$$ = nil
	}
	| import_list import
	{
		if debugParser {
			fmt.Println("import_list", $1, $2)
		}
		$$ = append($$, $2)
	}

imports:
	// Empty
	{ 
		$$ = nil
	}
	| IMPORT '(' import_list ')'
	{
		if debugParser {
			fmt.Println("imports", $3)
		}
		$$ = append($$, $3...)
	}

// options

options:
	// Empty
	{ 
		$$ = nil
	}
	| OPTIONS '(' option_list ')'
	{
		if debugParser {
			fmt.Println("options", $3)
		}
		$$ = append($$, $3...)
	}

option_list:
	// Empty
	{ 
		$$ = nil
	}
	| option_list option
	{
		if debugParser {
			fmt.Println("option_list", $1, $2)
		}
		$$ = append($$, $2)
	}

option:
	IDENT '=' STRING
	{
		if debugParser {
			fmt.Println("option ", $1, $3)
		}
		$$ = &ast.Option{
			Name:  $1,
			Value: trimString($3),
		}
	}


// definition

definition: 
	enum 
	| message
	| struct

definitions:
	// Empty
	{ 
		$$ = nil
	}
	| definitions definition
	{
		if debugParser {
			fmt.Println("definitions", $1, $2)
		}
		$$ = append($$, $2)
	}


// enum

enum: ENUM IDENT '{' enum_values '}'
	{
		if debugParser {
			fmt.Println("enum", $2, $4)
		}
		$$ = &ast.Definition{
			Type: ast.DefinitionEnum,
			Name: $2,

			Enum: &ast.Enum{
				Values: $4,
			},
		}
	}

enum_value: IDENT '=' INTEGER ';'
	{
		if debugParser {
			fmt.Println("enum value", $1, $3)
		}
		$$ = &ast.EnumValue{
			Name: $1,
			Value: $3,
		}
	}

enum_values:
	// Empty
	{
		$$ = nil
	}
	| enum_values enum_value
	{
		if debugParser {
			fmt.Println("enum values", $1, $2)
		}
		$$ = append($$, $2)
	}


// message

message: MESSAGE IDENT '{' message_fields '}' 
	{ 
		if debugParser {
			fmt.Println("message", $2, $4)
		}
		$$ = &ast.Definition{
			Type: ast.DefinitionMessage,
			Name: $2,

			Message: &ast.Message{
				Fields: $4,
			},
		}
	}

message_field: message_field_name type INTEGER ';'
	{
		if debugParser {
			fmt.Println("message field", $1, $2, $3)
		}
		$$ = &ast.MessageField{
			Name: $1,
			Type: $2,
			Tag: $3,
		}
	}

message_field_name:
	keyword
	| IDENT
	;

message_fields:
	// Empty
	{
		$$ = nil
	}
	| message_fields message_field
	{
		if debugParser {
			fmt.Println("message fields", $1, $2)
		}
		$$ = append($$, $2)
	}

// struct

struct: STRUCT IDENT '{' struct_fields '}' 
	{ 
		if debugParser {
			fmt.Println("struct", $2, $4)
		}
		$$ = &ast.Definition{
			Type: ast.DefinitionStruct,
			Name: $2,

			Struct: &ast.Struct{
				Fields: $4,
			},
		}
	}

struct_field: struct_field_name type ';'
	{
		if debugParser {
			fmt.Println("struct field", $1, $2)
		}
		$$ = &ast.StructField{
			Name: $1,
			Type: $2,
		}
	}

struct_field_name:
	keyword
	| IDENT
	;

struct_fields:
	// Empty
	{ 
		$$ = nil
	}
	| struct_fields struct_field
	{
		if debugParser {
			fmt.Println("struct fields", $1, $2)
		}
		$$ = append($$, $2)
	}

// type

type:
	base_type
	{
		if debugParser {
			fmt.Printf("type *%v\n", $1)
		}
		$$ = $1
	}	
	| '[' ']' base_type
	{
		if debugParser {
			fmt.Printf("type []%v\n", $3)
		}
		$$ = &ast.Type{
			Kind:    ast.KindList,
			Element: $3,
		}
	}

base_type:
	IDENT
	{
		if debugParser {
			fmt.Println("base type", $1)
		}
		$$ = &ast.Type{
			Kind: ast.GetKind($1),
			Name: $1,
		}
	}
	| IDENT '.' IDENT
	{
		if debugParser {
			fmt.Printf("base type %v.%v\n", $1, $3)
		}
		$$ = &ast.Type{
			Kind:   ast.KindReference,
			Name:   $3,
			Import: $1,
		}
	}
	| MESSAGE
	{
		if debugParser {
			fmt.Println("base type", $1)
		}
		$$ = &ast.Type{
			Kind: ast.KindAnyMessage,
			Name: "message",
		}
	}
