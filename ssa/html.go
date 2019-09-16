// Copyright 2015 The Go Authors. All rights reserved.
// Copyright 2019 Dominik Honnef. All rights reserved.

package ssa

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func live(f *Function) []bool {
	max := 0
	var ops []*Value

	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			if int(instr.ID()) > max {
				max = int(instr.ID())
			}
		}
	}

	out := make([]bool, max+1)
	var q []Node
	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			switch instr.(type) {
			case *Store, *Call, *Go, *Defer, *Return, *BlankStore, *Jump, *If:
				out[instr.ID()] = true
				q = append(q, instr)
			}
		}
	}

	for len(q) > 0 {
		v := q[len(q)-1]
		q = q[:len(q)-1]
		for _, op := range v.Operands(ops) {
			if *op == nil {
				continue
			}
			if !out[(*op).ID()] {
				out[(*op).ID()] = true
				q = append(q, *op)
			}
		}
	}

	return out
}

type funcPrinter interface {
	startBlock(b *BasicBlock, reachable bool)
	endBlock(b *BasicBlock)
	value(v Node, live bool)
	startDepCycle()
	endDepCycle()
	named(n *Parameter, vals []Value)
}

func fprintFunc(p funcPrinter, f *Function) {
	// XXX does our SSA form preserve unreachable blocks?
	// reachable, live := findlive(f)
	// XXX
	/*
		defer f.retDeadcodeLive(live)
	*/

	l := live(f)
	for _, b := range f.Blocks {
		// XXX
		// p.startBlock(b, reachable[b.Index])
		p.startBlock(b, true)

		end := len(b.Instrs) - 1
		if end < 0 {
			end = 0
		}
		for _, v := range b.Instrs[:end] {
			if _, ok := v.(*DebugRef); !ok {
				p.value(v, l[v.ID()])
			}
		}
		p.endBlock(b)
	}

	// XXX we don't have names, but we do have DebugRef
	/*
		for _, name := range f.Names {
			p.named(name, f.NamedValues[name])
		}
	*/
}

func opName(v Node) string {
	return reflect.ValueOf(v).Type().Elem().Name()
}

type HTMLWriter struct {
	w    io.WriteCloser
	path string
}

func NewHTMLWriter(path string, funcname, cfgMask string) *HTMLWriter {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("%v", err)
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("%v", err)
	}
	html := HTMLWriter{w: out, path: filepath.Join(pwd, path)}
	html.start(funcname)
	return &html
}

func (w *HTMLWriter) start(name string) {
	if w == nil {
		return
	}
	w.WriteString("<html>")
	w.WriteString(`<head>
<meta http-equiv="Content-Type" content="text/html;charset=UTF-8">
<style>

body {
    font-size: 14px;
    font-family: Arial, sans-serif;
}

h1 {
    font-size: 18px;
    display: inline-block;
    margin: 0 1em .5em 0;
}

#helplink {
    display: inline-block;
}

#help {
    display: none;
}

.stats {
    font-size: 60%;
}

table {
    border: 1px solid black;
    table-layout: fixed;
    width: 300px;
}

th, td {
    border: 1px solid black;
    overflow: hidden;
    width: 400px;
    vertical-align: top;
    padding: 5px;
}

td > h2 {
    cursor: pointer;
    font-size: 120%;
}

td.collapsed {
    font-size: 12px;
    width: 12px;
    border: 0px;
    padding: 0;
    cursor: pointer;
    background: #fafafa;
}

td.collapsed  div {
     -moz-transform: rotate(-90.0deg);  /* FF3.5+ */
       -o-transform: rotate(-90.0deg);  /* Opera 10.5 */
  -webkit-transform: rotate(-90.0deg);  /* Saf3.1+, Chrome */
             filter:  progid:DXImageTransform.Microsoft.BasicImage(rotation=0.083);  /* IE6,IE7 */
         -ms-filter: "progid:DXImageTransform.Microsoft.BasicImage(rotation=0.083)"; /* IE8 */
         margin-top: 10.3em;
         margin-left: -10em;
         margin-right: -10em;
         text-align: right;
}

code, pre, .lines, .ast {
    font-family: Menlo, monospace;
    font-size: 12px;
}

pre {
    -moz-tab-size: 4;
    -o-tab-size:   4;
    tab-size:      4;
}

.allow-x-scroll {
    overflow-x: scroll;
}

.lines {
    float: left;
    overflow: hidden;
    text-align: right;
}

.lines div {
    padding-right: 10px;
    color: gray;
}

div.line-number {
    font-size: 12px;
}

.ast {
    white-space: nowrap;
}

td.ssa-prog {
    width: 600px;
    word-wrap: break-word;
}

li {
    list-style-type: none;
}

li.ssa-long-value {
    text-indent: -2em;  /* indent wrapped lines */
}

li.ssa-value-list {
    display: inline;
}

li.ssa-start-block {
    padding: 0;
    margin: 0;
}

li.ssa-end-block {
    padding: 0;
    margin: 0;
}

ul.ssa-print-func {
    padding-left: 0;
}

li.ssa-start-block button {
    padding: 0 1em;
    margin: 0;
    border: none;
    display: inline;
    font-size: 14px;
    float: right;
}

button:hover {
    background-color: #eee;
    cursor: pointer;
}

dl.ssa-gen {
    padding-left: 0;
}

dt.ssa-prog-src {
    padding: 0;
    margin: 0;
    float: left;
    width: 4em;
}

dd.ssa-prog {
    padding: 0;
    margin-right: 0;
    margin-left: 4em;
}

.dead-value {
    color: gray;
}

.dead-block {
    opacity: 0.5;
}

.depcycle {
    font-style: italic;
}

.line-number {
    font-size: 11px;
}

.no-line-number {
    font-size: 11px;
    color: gray;
}

.zoom {
	position: absolute;
	float: left;
	white-space: nowrap;
	background-color: #eee;
}

.zoom a:link, .zoom a:visited  {
    text-decoration: none;
    color: blue;
    font-size: 16px;
    padding: 4px 2px;
}

svg {
    cursor: default;
    outline: 1px solid #eee;
}

.highlight-aquamarine     { background-color: aquamarine; }
.highlight-coral          { background-color: coral; }
.highlight-lightpink      { background-color: lightpink; }
.highlight-lightsteelblue { background-color: lightsteelblue; }
.highlight-palegreen      { background-color: palegreen; }
.highlight-skyblue        { background-color: skyblue; }
.highlight-lightgray      { background-color: lightgray; }
.highlight-yellow         { background-color: yellow; }
.highlight-lime           { background-color: lime; }
.highlight-khaki          { background-color: khaki; }
.highlight-aqua           { background-color: aqua; }
.highlight-salmon         { background-color: salmon; }

.outline-blue           { outline: blue solid 2px; }
.outline-red            { outline: red solid 2px; }
.outline-blueviolet     { outline: blueviolet solid 2px; }
.outline-darkolivegreen { outline: darkolivegreen solid 2px; }
.outline-fuchsia        { outline: fuchsia solid 2px; }
.outline-sienna         { outline: sienna solid 2px; }
.outline-gold           { outline: gold solid 2px; }
.outline-orangered      { outline: orangered solid 2px; }
.outline-teal           { outline: teal solid 2px; }
.outline-maroon         { outline: maroon solid 2px; }
.outline-black          { outline: black solid 2px; }

ellipse.outline-blue           { stroke-width: 2px; stroke: blue; }
ellipse.outline-red            { stroke-width: 2px; stroke: red; }
ellipse.outline-blueviolet     { stroke-width: 2px; stroke: blueviolet; }
ellipse.outline-darkolivegreen { stroke-width: 2px; stroke: darkolivegreen; }
ellipse.outline-fuchsia        { stroke-width: 2px; stroke: fuchsia; }
ellipse.outline-sienna         { stroke-width: 2px; stroke: sienna; }
ellipse.outline-gold           { stroke-width: 2px; stroke: gold; }
ellipse.outline-orangered      { stroke-width: 2px; stroke: orangered; }
ellipse.outline-teal           { stroke-width: 2px; stroke: teal; }
ellipse.outline-maroon         { stroke-width: 2px; stroke: maroon; }
ellipse.outline-black          { stroke-width: 2px; stroke: black; }

</style>

<script type="text/javascript">
// ordered list of all available highlight colors
var highlights = [
    "highlight-aquamarine",
    "highlight-coral",
    "highlight-lightpink",
    "highlight-lightsteelblue",
    "highlight-palegreen",
    "highlight-skyblue",
    "highlight-lightgray",
    "highlight-yellow",
    "highlight-lime",
    "highlight-khaki",
    "highlight-aqua",
    "highlight-salmon"
];

// state: which value is highlighted this color?
var highlighted = {};
for (var i = 0; i < highlights.length; i++) {
    highlighted[highlights[i]] = "";
}

// ordered list of all available outline colors
var outlines = [
    "outline-blue",
    "outline-red",
    "outline-blueviolet",
    "outline-darkolivegreen",
    "outline-fuchsia",
    "outline-sienna",
    "outline-gold",
    "outline-orangered",
    "outline-teal",
    "outline-maroon",
    "outline-black"
];

// state: which value is outlined this color?
var outlined = {};
for (var i = 0; i < outlines.length; i++) {
    outlined[outlines[i]] = "";
}

window.onload = function() {
    var ssaElemClicked = function(elem, event, selections, selected) {
        event.stopPropagation();

        // TODO: pushState with updated state and read it on page load,
        // so that state can survive across reloads

        // find all values with the same name
        var c = elem.classList.item(0);
        var x = document.getElementsByClassName(c);

        // if selected, remove selections from all of them
        // otherwise, attempt to add

        var remove = "";
        for (var i = 0; i < selections.length; i++) {
            var color = selections[i];
            if (selected[color] == c) {
                remove = color;
                break;
            }
        }

        if (remove != "") {
            for (var i = 0; i < x.length; i++) {
                x[i].classList.remove(remove);
            }
            selected[remove] = "";
            return;
        }

        // we're adding a selection
        // find first available color
        var avail = "";
        for (var i = 0; i < selections.length; i++) {
            var color = selections[i];
            if (selected[color] == "") {
                avail = color;
                break;
            }
        }
        if (avail == "") {
            alert("out of selection colors; go add more");
            return;
        }

        // set that as the selection
        for (var i = 0; i < x.length; i++) {
            x[i].classList.add(avail);
        }
        selected[avail] = c;
    };

    var ssaValueClicked = function(event) {
        ssaElemClicked(this, event, highlights, highlighted);
    };

    var ssaBlockClicked = function(event) {
        ssaElemClicked(this, event, outlines, outlined);
    };

    var ssavalues = document.getElementsByClassName("ssa-value");
    for (var i = 0; i < ssavalues.length; i++) {
        ssavalues[i].addEventListener('click', ssaValueClicked);
    }

    var ssalongvalues = document.getElementsByClassName("ssa-long-value");
    for (var i = 0; i < ssalongvalues.length; i++) {
        // don't attach listeners to li nodes, just the spans they contain
        if (ssalongvalues[i].nodeName == "SPAN") {
            ssalongvalues[i].addEventListener('click', ssaValueClicked);
        }
    }

    var ssablocks = document.getElementsByClassName("ssa-block");
    for (var i = 0; i < ssablocks.length; i++) {
        ssablocks[i].addEventListener('click', ssaBlockClicked);
    }

    var lines = document.getElementsByClassName("line-number");
    for (var i = 0; i < lines.length; i++) {
        lines[i].addEventListener('click', ssaValueClicked);
    }

    // Contains phase names which are expanded by default. Other columns are collapsed.
    var expandedDefault = [
        "start",
        "deadcode",
        "opt",
        "lower",
        "late deadcode",
        "regalloc",
        "genssa",
    ];

    function toggler(phase) {
        return function() {
            toggle_cell(phase+'-col');
            toggle_cell(phase+'-exp');
        };
    }

    function toggle_cell(id) {
        var e = document.getElementById(id);
        if (e.style.display == 'table-cell') {
            e.style.display = 'none';
        } else {
            e.style.display = 'table-cell';
        }
    }

    // Go through all columns and collapse needed phases.
    var td = document.getElementsByTagName("td");
    for (var i = 0; i < td.length; i++) {
        var id = td[i].id;
        var phase = id.substr(0, id.length-4);
        var show = expandedDefault.indexOf(phase) !== -1
        if (id.endsWith("-exp")) {
            var h2 = td[i].getElementsByTagName("h2");
            if (h2 && h2[0]) {
                h2[0].addEventListener('click', toggler(phase));
            }
        } else {
            td[i].addEventListener('click', toggler(phase));
        }
        if (id.endsWith("-col") && show || id.endsWith("-exp") && !show) {
            td[i].style.display = 'none';
            continue;
        }
        td[i].style.display = 'table-cell';
    }

    // find all svg block nodes, add their block classes
    var nodes = document.querySelectorAll('*[id^="graph_node_"]');
    for (var i = 0; i < nodes.length; i++) {
    	var node = nodes[i];
    	var name = node.id.toString();
    	var block = name.substring(name.lastIndexOf("_")+1);
    	node.classList.remove("node");
    	node.classList.add(block);
        node.addEventListener('click', ssaBlockClicked);
        var ellipse = node.getElementsByTagName('ellipse')[0];
        ellipse.classList.add(block);
        ellipse.addEventListener('click', ssaBlockClicked);
    }

    // make big graphs smaller
    var targetScale = 0.5;
    var nodes = document.querySelectorAll('*[id^="svg_graph_"]');
    // TODO: Implement smarter auto-zoom using the viewBox attribute
    // and in case of big graphs set the width and height of the svg graph to
    // maximum allowed.
    for (var i = 0; i < nodes.length; i++) {
    	var node = nodes[i];
    	var name = node.id.toString();
    	var phase = name.substring(name.lastIndexOf("_")+1);
    	var gNode = document.getElementById("g_graph_"+phase);
    	var scale = gNode.transform.baseVal.getItem(0).matrix.a;
    	if (scale > targetScale) {
    		node.width.baseVal.value *= targetScale / scale;
    		node.height.baseVal.value *= targetScale / scale;
    	}
    }
};

function toggle_visibility(id) {
    var e = document.getElementById(id);
    if (e.style.display == 'block') {
        e.style.display = 'none';
    } else {
        e.style.display = 'block';
    }
}

function hideBlock(el) {
    var es = el.parentNode.parentNode.getElementsByClassName("ssa-value-list");
    if (es.length===0)
        return;
    var e = es[0];
    if (e.style.display === 'block' || e.style.display === '') {
        e.style.display = 'none';
        el.innerHTML = '+';
    } else {
        e.style.display = 'block';
        el.innerHTML = '-';
    }
}

// TODO: scale the graph with the viewBox attribute.
function graphReduce(id) {
    var node = document.getElementById(id);
    if (node) {
    		node.width.baseVal.value *= 0.9;
    		node.height.baseVal.value *= 0.9;
    }
    return false;
}

function graphEnlarge(id) {
    var node = document.getElementById(id);
    if (node) {
    		node.width.baseVal.value *= 1.1;
    		node.height.baseVal.value *= 1.1;
    }
    return false;
}

function makeDraggable(event) {
    var svg = event.target;
    if (window.PointerEvent) {
        svg.addEventListener('pointerdown', startDrag);
        svg.addEventListener('pointermove', drag);
        svg.addEventListener('pointerup', endDrag);
        svg.addEventListener('pointerleave', endDrag);
    } else {
        svg.addEventListener('mousedown', startDrag);
        svg.addEventListener('mousemove', drag);
        svg.addEventListener('mouseup', endDrag);
        svg.addEventListener('mouseleave', endDrag);
    }

    var point = svg.createSVGPoint();
    var isPointerDown = false;
    var pointerOrigin;
    var viewBox = svg.viewBox.baseVal;

    function getPointFromEvent (event) {
        point.x = event.clientX;
        point.y = event.clientY;

        // We get the current transformation matrix of the SVG and we inverse it
        var invertedSVGMatrix = svg.getScreenCTM().inverse();
        return point.matrixTransform(invertedSVGMatrix);
    }

    function startDrag(event) {
        isPointerDown = true;
        pointerOrigin = getPointFromEvent(event);
    }

    function drag(event) {
        if (!isPointerDown) {
            return;
        }
        event.preventDefault();

        var pointerPosition = getPointFromEvent(event);
        viewBox.x -= (pointerPosition.x - pointerOrigin.x);
        viewBox.y -= (pointerPosition.y - pointerOrigin.y);
    }

    function endDrag(event) {
        isPointerDown = false;
    }
}</script>

</head>`)
	w.WriteString("<body>")
	w.WriteString("<h1>")
	w.WriteString(html.EscapeString(name))
	w.WriteString("</h1>")
	w.WriteString(`
<a href="#" onclick="toggle_visibility('help');return false;" id="helplink">help</a>
<div id="help">

<p>
Click on a value or block to toggle highlighting of that value/block
and its uses.  (Values and blocks are highlighted by ID, and IDs of
dead items may be reused, so not all highlights necessarily correspond
to the clicked item.)
</p>

<p>
Faded out values and blocks are dead code that has not been eliminated.
</p>

<p>
Values printed in italics have a dependency cycle.
</p>

<p>
<b>CFG</b>: Dashed edge is for unlikely branches. Blue color is for backward edges.
Edge with a dot means that this edge follows the order in which blocks were laidout.
</p>

</div>
`)
	w.WriteString("<table>")
	w.WriteString("<tr>")
}

func (w *HTMLWriter) Close() {
	if w == nil {
		return
	}
	io.WriteString(w.w, "</tr>")
	io.WriteString(w.w, "</table>")
	io.WriteString(w.w, "</body>")
	io.WriteString(w.w, "</html>")
	w.w.Close()
	fmt.Printf("dumped SSA to %v\n", w.path)
}

// WriteFunc writes f in a column headed by title.
// phase is used for collapsing columns and should be unique across the table.
func (w *HTMLWriter) WriteFunc(phase, title string, f *Function) {
	if w == nil {
		return
	}
	w.WriteColumn(phase, title, "", FuncHTML(f, phase))
}

// WriteColumn writes raw HTML in a column headed by title.
// It is intended for pre- and post-compilation log output.
func (w *HTMLWriter) WriteColumn(phase, title, class, html string) {
	if w == nil {
		return
	}
	id := strings.Replace(phase, " ", "-", -1)
	// collapsed column
	w.Printf("<td id=\"%v-col\" class=\"collapsed\"><div>%v</div></td>", id, phase)

	if class == "" {
		w.Printf("<td id=\"%v-exp\">", id)
	} else {
		w.Printf("<td id=\"%v-exp\" class=\"%v\">", id, class)
	}
	w.WriteString("<h2>" + title + "</h2>")
	w.WriteString(html)
	w.WriteString("</td>")
}

func (w *HTMLWriter) Printf(msg string, v ...interface{}) {
	if _, err := fmt.Fprintf(w.w, msg, v...); err != nil {
		log.Fatalf("%v", err)
	}
}

func (w *HTMLWriter) WriteString(s string) {
	if _, err := io.WriteString(w.w, s); err != nil {
		log.Fatalf("%v", err)
	}
}

func ValueHTML(v Node) string {
	if v == nil {
		return "&lt;nil&gt;"
	}
	// TODO: Using the value ID as the class ignores the fact
	// that value IDs get recycled and that some values
	// are transmuted into other values.
	class := fmt.Sprintf("t%d", v.ID())
	var label string
	switch v := v.(type) {
	case *Function:
		label = v.Name()
	case *Builtin:
		label = v.Name()
	default:
		label = class
	}
	return fmt.Sprintf("<span class=\"%s ssa-value\">%s</span>", class, label)
}

func ValueLongHTML(v Node) string {
	// TODO: Any intra-value formatting?
	// I'm wary of adding too much visual noise,
	// but a little bit might be valuable.
	// We already have visual noise in the form of punctuation
	// maybe we could replace some of that with formatting.
	s := fmt.Sprintf("<span class=\"t%d ssa-long-value\">", v.ID())

	// XXX don't bother with line numbers
	linenumber := "<span class=\"no-line-number\">(?)</span>"
	s += fmt.Sprintf("%s %s = %s", ValueHTML(v), linenumber, opName(v))

	if v, ok := v.(Value); ok {
		s += " &lt;" + html.EscapeString(v.Type().String()) + "&gt;"
	}

	switch v := v.(type) {
	case *Parameter:
		s += fmt.Sprintf(" {%s}", html.EscapeString(v.name))
	case *BinOp:
		s += fmt.Sprintf(" {%s}", html.EscapeString(v.Op.String()))
	case *UnOp:
		s += fmt.Sprintf(" {%s}", html.EscapeString(v.Op.String()))
	case *Const:
		if v.Value == nil {
			s += " {&lt;nil&gt;}"
		} else {
			s += fmt.Sprintf(" {%s}", html.EscapeString(v.Value.String()))
		}
	}
	for _, a := range v.Operands(nil) {
		s += fmt.Sprintf(" %s", ValueHTML(*a))
	}

	// XXX we don't have NamedValues, but we have DebugRefs
	/*
		var names []string
		for name, values := range v.Block.Func.NamedValues {
			for _, value := range values {
				if value == v {
					names = append(names, name.String())
					break // drop duplicates.
				}
			}
		}
		if len(names) != 0 {
			s += " (" + strings.Join(names, ", ") + ")"
		}
	*/

	s += "</span>"
	return s
}

func BlockHTML(b *BasicBlock) string {
	// TODO: Using the value ID as the class ignores the fact
	// that value IDs get recycled and that some values
	// are transmuted into other values.
	s := html.EscapeString(b.String())
	return fmt.Sprintf("<span class=\"%s ssa-block\">%s</span>", s, s)
}

func BlockLongHTML(b *BasicBlock) string {
	var kind string
	var term Instruction
	if len(b.Instrs) > 0 {
		term = b.Instrs[len(b.Instrs)-1]
		kind = opName(term)
	}
	// TODO: improve this for HTML?
	s := fmt.Sprintf("<span class=\"b%d ssa-block\">%s</span>", b.Index, kind)
	// XXX
	/*
		if b.Aux != nil {
			s += html.EscapeString(fmt.Sprintf(" {%v}", b.Aux))
		}
	*/
	// XXX we don't have Control, we need to look at the last
	// instruction in the block. we probably shouldn't print the last
	// instruction in the value list, either.
	/*
		if b.Control != nil {
			s += fmt.Sprintf(" %s", b.Control.HTML())
		}
	*/
	if term != nil {
		ops := term.Operands(nil)
		if len(ops) > 0 {
			var ss []string
			for _, op := range ops {
				ss = append(ss, ValueHTML(*op))
			}
			s += " " + strings.Join(ss, ", ")
		}
	}
	if len(b.Succs) > 0 {
		s += " &#8594;" // right arrow
		for _, c := range b.Succs {
			s += " " + BlockHTML(c)
		}
	}
	return s
}

func FuncHTML(f *Function, phase string) string {
	buf := new(bytes.Buffer)
	fmt.Fprint(buf, "<code>")
	p := htmlFuncPrinter{w: buf}
	fprintFunc(p, f)

	// fprintFunc(&buf, f) // TODO: HTML, not text, <br /> for line breaks, etc.
	fmt.Fprint(buf, "</code>")
	return buf.String()
}

type htmlFuncPrinter struct {
	w io.Writer
}

func (p htmlFuncPrinter) startBlock(b *BasicBlock, reachable bool) {
	var dead string
	if !reachable {
		dead = "dead-block"
	}
	fmt.Fprintf(p.w, "<ul class=\"%s ssa-print-func %s\">", b, dead)
	fmt.Fprintf(p.w, "<li class=\"ssa-start-block\">%s:", BlockHTML(b))
	if len(b.Preds) > 0 {
		io.WriteString(p.w, " &#8592;") // left arrow
		for _, pred := range b.Preds {
			fmt.Fprintf(p.w, " %s", BlockHTML(pred))
		}
	}
	if len(b.Instrs) > 0 {
		io.WriteString(p.w, `<button onclick="hideBlock(this)">-</button>`)
	}
	io.WriteString(p.w, "</li>")
	if len(b.Instrs) > 0 { // start list of values
		io.WriteString(p.w, "<li class=\"ssa-value-list\">")
		io.WriteString(p.w, "<ul>")
	}
}

func (p htmlFuncPrinter) endBlock(b *BasicBlock) {
	if len(b.Instrs) > 0 { // end list of values
		io.WriteString(p.w, "</ul>")
		io.WriteString(p.w, "</li>")
	}
	io.WriteString(p.w, "<li class=\"ssa-end-block\">")
	fmt.Fprint(p.w, BlockLongHTML(b))
	io.WriteString(p.w, "</li>")
	io.WriteString(p.w, "</ul>")
}

func (p htmlFuncPrinter) value(v Node, live bool) {
	var dead string
	if !live {
		dead = "dead-value"
	}
	fmt.Fprintf(p.w, "<li class=\"ssa-long-value %s\">", dead)
	fmt.Fprint(p.w, ValueLongHTML(v))
	io.WriteString(p.w, "</li>")
}

func (p htmlFuncPrinter) startDepCycle() {
	fmt.Fprintln(p.w, "<span class=\"depcycle\">")
}

func (p htmlFuncPrinter) endDepCycle() {
	fmt.Fprintln(p.w, "</span>")
}

func (p htmlFuncPrinter) named(n *Parameter, vals []Value) {
	fmt.Fprintf(p.w, "<li>name %s: ", n)
	for _, val := range vals {
		fmt.Fprintf(p.w, "%s ", ValueHTML(val))
	}
	fmt.Fprintf(p.w, "</li>")
}
