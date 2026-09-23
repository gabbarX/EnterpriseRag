package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFirstBrokenPythonQuoteCatchesNestedASCIIQuotes(t *testing.T) {
	src := "slides = [\n" +
		"    (\"यह रात नहीं\"जल्दबाजी\"वाली, बल्कि सोचने की\", False),\n" +
		"]\n"
	line, ok := firstBrokenPythonQuote(src)
	require.True(t, ok)
	assert.Equal(t, 2, line)
}

func TestFirstBrokenPythonQuoteAllowsKeywordAfterString(t *testing.T) {
	_, ok := firstBrokenPythonQuote(`x = "hello" if cond else "bye"`)
	assert.False(t, ok)
}

func TestFirstBrokenPythonQuoteAllowsImplicitConcat(t *testing.T) {
	_, ok := firstBrokenPythonQuote("x = \"hello\" \"world\"\n")
	assert.False(t, ok)
}

func TestFirstBrokenPythonQuoteAllowsSingleQuotedDevanagari(t *testing.T) {
	_, ok := firstBrokenPythonQuote("x = 'यह रात नहीं\"जल्दबाजी\"वाली'\n")
	assert.False(t, ok)
}

func TestFirstBrokenPythonQuoteAllowsCurlyQuotes(t *testing.T) {
	_, ok := firstBrokenPythonQuote("x = \"यह रात नहीं“जल्दबाजी”वाली\"\n")
	assert.False(t, ok)
}

func TestFirstBrokenPythonQuoteAllowsFString(t *testing.T) {
	_, ok := firstBrokenPythonQuote("x = f\"hello {name}\"\n")
	assert.False(t, ok)
}

func TestFirstBrokenPythonQuoteReportsUnclosed(t *testing.T) {
	line, ok := firstBrokenPythonQuote("x = \"hello\n")
	require.True(t, ok)
	assert.Equal(t, 1, line)
}

func TestPythonScriptSyntaxHintOnlyForPy(t *testing.T) {
	src := "(\"यह रात नहीं\"जल्दबाजी\"वाली\")\n"
	assert.Contains(t, pythonScriptSyntaxHint("/workspace/output/x.py", src, ToolEditSandboxFile),
		ToolEditSandboxFile, "the hint must name the tool that can repair this file")
	assert.Contains(t, pythonScriptSyntaxHint(testSkillDir+"/run.py", src, ToolEditSkillFile),
		ToolEditSkillFile, "a skill-tree file is not repairable with edit_sandbox_file")
	assert.Empty(t, pythonScriptSyntaxHint("/workspace/output/x.md", src, ToolEditSandboxFile))
}

func TestPythonSyntaxErrorHint(t *testing.T) {
	assert.Empty(t, pythonSyntaxErrorHint("ModuleNotFoundError: docx"))
	assert.Contains(t, pythonSyntaxErrorHint("SyntaxError: invalid syntax"), "edit_sandbox_file")
}
