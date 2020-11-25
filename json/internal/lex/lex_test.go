package lex_test

import (
	"strings"
	"testing"

	"github.com/bounoable/dragoman/json/internal/lex"
	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []lex.Token
	}{
		{
			name:  "empty file",
			input: ``,
			expected: []lex.Token{
				{Type: lex.EOF, Pos: 0},
			},
		},
		{
			name:  "empty file with whitespace",
			input: `   `,
			expected: []lex.Token{
				{Type: lex.EOF, Pos: 3},
			},
		},
		{
			name:  "empty object",
			input: `{}`,
			expected: []lex.Token{
				{Type: lex.EOF, Pos: 2},
			},
		},
		{
			name:  "empty array",
			input: `[]`,
			expected: []lex.Token{
				{Type: lex.EOF, Pos: 2},
			},
		},
		{
			name:  "null",
			input: `null`,
			expected: []lex.Token{
				{Type: lex.EOF, Pos: 4},
			},
		},
		{
			name:  "string, well-formed",
			input: `"This is a test."`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 0, Value: `"This is a test."`},
				{Type: lex.EOF, Pos: 17},
			},
		},
		{
			name:  "string, with quotes",
			input: `"This \" is a \"test\"."`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 0, Value: `"This \" is a \"test\"."`},
				{Type: lex.EOF, Pos: 24},
			},
		},
		{
			name:  "integer",
			input: `-1738`,
			expected: []lex.Token{
				{Type: lex.EOF, Pos: 5},
			},
		},
		{
			name:  "float",
			input: `-17.38`,
			expected: []lex.Token{
				{Type: lex.EOF, Pos: 6},
			},
		},
		{
			name:  "flat object, well-formed",
			input: `{"title": "This is a title.", "description": "This is a description."}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 10, Value: `"This is a title."`},
				{Type: lex.String, Pos: 45, Value: `"This is a description."`},
				{Type: lex.EOF, Pos: 70},
			},
		},
		{
			name:  "flat object, ugly",
			input: `{"title"   :   "This is a title.", "description"      :"This is a description."}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 15, Value: `"This is a title."`},
				{Type: lex.String, Pos: 55, Value: `"This is a description."`},
				{Type: lex.EOF, Pos: 80},
			},
		},
		{
			name:  "flat object, with quotes",
			input: `{"\"title\"": "This is a title.", "description": "This is a \"description\"."}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 14, Value: `"This is a title."`},
				{Type: lex.String, Pos: 49, Value: `"This is a \"description\"."`},
				{Type: lex.EOF, Pos: 78},
			},
		},
		{
			name: "nested object, well-formed",
			input: `{
				"nested1": {"title": "This is the first title.", "description": "This is the first description."},
				"nested2": {"title": "This is the second title.", "description": "This is the second description."},
			}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 27, Value: `"This is the first title."`},
				{Type: lex.String, Pos: 70, Value: `"This is the first description."`},
				{Type: lex.String, Pos: 130, Value: `"This is the second title."`},
				{Type: lex.String, Pos: 174, Value: `"This is the second description."`},
				{Type: lex.EOF, Pos: 214},
			},
		},
		{
			name: "nested object, ugly",
			input: `{
				   "nested1"  : {  "title": "This is the first title."  ,"description":"This is the first description."   },
				"nested2":   {  "title"   :"This is the second title.", "description" : "This is the second description."  },
			}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 34, Value: `"This is the first title."`},
				{Type: lex.String, Pos: 77, Value: `"This is the first description."`},
				{Type: lex.String, Pos: 146, Value: `"This is the second title."`},
				{Type: lex.String, Pos: 191, Value: `"This is the second description."`},
				{Type: lex.EOF, Pos: 233},
			},
		},
		{
			name: "nested object, with quotes",
			input: `{
				"nested1": {"title": "This is the \"first\" title.", "description": "This is the first \\description\\."},
				"nested2": {"\"title\"": "This is the \\second\\ title.", "description": "This is the second \"description\"."},
			}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 27, Value: `"This is the \"first\" title."`},
				{Type: lex.String, Pos: 74, Value: `"This is the first \\description\\."`},
				{Type: lex.String, Pos: 142, Value: `"This is the \\second\\ title."`},
				{Type: lex.String, Pos: 190, Value: `"This is the second \"description\"."`},
				{Type: lex.EOF, Pos: 234},
			},
		},
		{
			name:  "flat array, well-formed",
			input: `["Hello", "Bye", "How are you?"]`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 1, Value: `"Hello"`},
				{Type: lex.String, Pos: 10, Value: `"Bye"`},
				{Type: lex.String, Pos: 17, Value: `"How are you?"`},
				{Type: lex.EOF, Pos: 32},
			},
		},
		{
			name:  "flat array, ugly",
			input: `[   "Hello",   "Bye"   ,"How are you?"    ]`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 4, Value: `"Hello"`},
				{Type: lex.String, Pos: 15, Value: `"Bye"`},
				{Type: lex.String, Pos: 24, Value: `"How are you?"`},
				{Type: lex.EOF, Pos: 43},
			},
		},
		{
			name:  "flat array, with quotes",
			input: `["Hello", "\"Bye\"", "How \"are\\ you?"]`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 1, Value: `"Hello"`},
				{Type: lex.String, Pos: 10, Value: `"\"Bye\""`},
				{Type: lex.String, Pos: 21, Value: `"How \"are\\ you?"`},
				{Type: lex.EOF, Pos: 40},
			},
		},
		{
			name: "array of objects, well-formed",
			input: `[
				{"name": "Bob", "age": 50},
				{"name": "Linda", "age": 45},
			]`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 15, Value: `"Bob"`},
				{Type: lex.String, Pos: 47, Value: `"Linda"`},
				{Type: lex.EOF, Pos: 72},
			},
		},
		{
			name: "array of objects, with quotes",
			input: `[
				{"name": "\"Bob\"", "age": 50},
				{"name": "\\Linda\\", "age": 45},
			]`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 15, Value: `"\"Bob\""`},
				{Type: lex.String, Pos: 51, Value: `"\\Linda\\"`},
				{Type: lex.EOF, Pos: 80},
			},
		},
		{
			name:  "nested object with array field, well-formed",
			input: `{"nested": {"collection": ["This is an item.", "This is another item."]}}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 27, Value: `"This is an item."`},
				{Type: lex.String, Pos: 47, Value: `"This is another item."`},
				{Type: lex.EOF, Pos: 73},
			},
		},
		{
			name: "nested object with mixed array, ugly",
			input: `{  "nested": {"under"  : {
				"person": {
					"bob"  :{
						"name": "Bob",

						"age": 37,

						"skills": ["cooking", "sleeping", -4.38, true, {
							"name"  : "jumping",
							"height":   1.2  ,
						}],

						"quotes": [
							"Hello.",
							"\"This\" is a word.",
							"You're all terrible."
						]
					}
				}
			}  }}`,
			expected: []lex.Token{
				{Type: lex.String, Pos: 72, Value: `"Bob"`},
				{Type: lex.String, Pos: 115, Value: `"cooking"`},
				{Type: lex.String, Pos: 126, Value: `"sleeping"`},
				{Type: lex.String, Pos: 170, Value: `"jumping"`},
				{Type: lex.String, Pos: 243, Value: `"Hello."`},
				{Type: lex.String, Pos: 260, Value: `"\"This\" is a word."`},
				{Type: lex.String, Pos: 290, Value: `"You're all terrible."`},
				{Type: lex.EOF, Pos: 342},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			items := lex.Lex(strings.NewReader(test.input))
			assert.Equal(t, test.expected, drain(items))
		})
	}
}

func drain(ch <-chan lex.Token) []lex.Token {
	var items []lex.Token
	for item := range ch {
		items = append(items, item)
	}
	return items
}
