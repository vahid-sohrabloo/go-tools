package edit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

func Fix(msg string, edits ...analysis.TextEdit) *analysis.SuggestedFix {
	return &analysis.SuggestedFix{
		Message:   msg,
		TextEdits: edits,
	}
}

func ReplaceNode(fset *token.FileSet, old, new ast.Node) analysis.TextEdit {
	w := &bytes.Buffer{}
	if err := format.Node(w, fset, new); err != nil {
		panic(fmt.Sprintf("internal error: %v", err))
	}
	return analysis.TextEdit{
		Pos:     old.Pos(),
		End:     old.End(),
		NewText: w.Bytes(),
	}
}

func ReplaceNodeWithText(old ast.Node, new string) analysis.TextEdit {
	return analysis.TextEdit{
		Pos:     old.Pos(),
		End:     old.End(),
		NewText: []byte(new),
	}
}

func DeleteNode(node ast.Node) analysis.TextEdit {
	return analysis.TextEdit{
		Pos:     node.Pos(),
		End:     node.End(),
		NewText: []byte{},
	}
}
