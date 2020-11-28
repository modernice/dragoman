package html_test

import (
	"context"
	"strings"
	"testing"
	"time"

	stdhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/bounoable/dragoman/format/html"
	"github.com/bounoable/dragoman/text"
	"github.com/stretchr/testify/assert"
)

func TestRanger_Ranges(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []text.Range
	}{
		{
			name: "only text",
			input: `This is a paragraph without tags.
					This is a another paragraph without tags.`,
			expected: []text.Range{
				{0, 80},
			},
		},
		{
			name: "only text with umlauts",
			input: `This is a pärägräph withöut tags.
					This is a anöther paragraph withöut tags.`,
			expected: []text.Range{
				{0, 80},
			},
		},
		{
			name: "simple paragraphs",
			input: `<p>This is a paragraph.</p>
					<p>This is another paragraph.</p>`,
			expected: []text.Range{
				{3, 23},
				{36, 62},
			},
		},
		{
			name: "simple spans",
			input: `<span>This is a span.</span>
					<span>This is another span.</span>`,
			expected: []text.Range{
				{6, 21},
				{40, 61},
			},
		},
		{
			name:  "paragraph with attributes",
			input: `<p attr1="I'm an attribute." attr2="Me too!">I am a paragraph.</p>`,
			expected: []text.Range{
				{45, 62},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testRanger(t, test.input, test.expected)
		})
	}
}

func TestRanger_Ranges_withAttributeFunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		fn       func(stdhtml.Token) []string
		expected []text.Range
	}{
		{
			name:  "single attribute",
			input: `<p alt="An alternate description.">A paragraph with an <img alt="An alternate description." src="/path/to/image.png">, goodbye.</p>`,
			fn: func(s stdhtml.Token) []string {
				if s.DataAtom != atom.Img {
					return nil
				}
				return []string{"alt"}
			},
			expected: []text.Range{
				{35, 55},
				{65, 90},
				{117, 127},
			},
		},
		{
			name:  "multiple attributes",
			input: `<p alt="An alternate description.">A paragraph with an <img alt="An alternate description." src="/path/to/image.png">, goodbye.</p>`,
			fn: func(stdhtml.Token) []string {
				return []string{"alt"}
			},
			expected: []text.Range{
				{8, 33},
				{35, 55},
				{65, 90},
				{117, 127},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testRanger(t, test.input, test.expected, html.WithAttributeFunc(test.fn))
		})
	}
}

func TestRanger_Ranges_withAttribute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     []html.Option
		expected []text.Range
	}{
		{
			name:  "single attribute, single tag",
			input: `<p alt="An alternate description.">A paragraph with an <img alt="An alternate description." src="/path/to/image.png">, goodbye.</p>`,
			opts: []html.Option{
				html.WithAttribute("alt", "img"),
			},
			expected: []text.Range{
				{35, 55},
				{65, 90},
				{117, 127},
			},
		},
		{
			name:  "multiple attributes, multiple tags",
			input: `<p alt="An alternate description.">A paragraph with an <img alt="An alternate description." src="/path/to/image.png">, goodbye.</p>`,
			opts: []html.Option{
				html.WithAttribute("alt", "img", "p"),
				html.WithAttribute("src", "img"),
			},
			expected: []text.Range{
				{8, 33},
				{35, 55},
				{65, 90},
				{97, 115},
				{117, 127},
			},
		},
		{
			name:  "multiple attributes, all tags",
			input: `<p alt="An alternate description.">A paragraph with an <img alt="An alternate description." src="/path/to/image.png">, goodbye.</p>`,
			opts: []html.Option{
				html.WithAttribute("alt"),
				html.WithAttribute("src"),
			},
			expected: []text.Range{
				{8, 33},
				{35, 55},
				{65, 90},
				{97, 115},
				{117, 127},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testRanger(t, test.input, test.expected, test.opts...)
		})
	}
}

func TestRanger_Ranges_withAttributePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		opts     []html.Option
		expected []text.Range
	}{
		{
			name:  "single attribute, single tag",
			input: `<p alt="An alternate description.">A paragraph with an <img alt="An alternate description." src="/path/to/image.png">, goodbye.</p>`,
			opts: []html.Option{
				optionMust(html.WithAttributePath("img.alt")),
			},
			expected: []text.Range{
				{35, 55},
				{65, 90},
				{117, 127},
			},
		},
		{
			name:  "multiple attributes, multiple tags",
			input: `<p alt="An alternate description.">A paragraph with an <img alt="An alternate description." src="/path/to/image.png">, goodbye.</p>`,
			opts: []html.Option{
				optionMust(html.WithAttributePath("img.alt", "p.alt")),
				optionMust(html.WithAttributePath("img.src")),
			},
			expected: []text.Range{
				{8, 33},
				{35, 55},
				{65, 90},
				{97, 115},
				{117, 127},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testRanger(t, test.input, test.expected, test.opts...)
		})
	}
}

func testRanger(t *testing.T, input string, expected []text.Range, opts ...html.Option) {
	ranger := html.Ranger(opts...)
	ch, _ := ranger.Ranges(context.Background(), strings.NewReader(input))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	ranges, err := drain(ctx, ch)
	assert.Nil(t, err)
	assert.Equal(t, expected, ranges)
}

func drain(ctx context.Context, ch <-chan text.Range) ([]text.Range, error) {
	var ranges []text.Range
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case rang, ok := <-ch:
			if !ok {
				return ranges, nil
			}
			ranges = append(ranges, rang)
		}
	}
}

func optionMust(opt html.Option, err error) html.Option {
	if err != nil {
		panic(err)
	}
	return opt
}
