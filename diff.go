package yiff

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Yiff holds necessary fields for diff
type Yiff struct {
	file1, file2    io.Reader
	preCtx, postCtx int
}

// New returns a new istance of Yiff with two reader
func New(f1, f2 io.Reader) *Yiff {
	return &Yiff{
		file1: f1,
		file2: f2,
	}
}

// Result holds result of the a diff
type Result struct {
	Line int
	*bytes.Buffer
}

// Results is the collection of multiple diff
type Results []Result

func printNode(suf string, n *yaml.Node) {
	fmt.Println(suf, n.Line, ":", n.Column, n.Kind, n.Tag, n.Value)
	for _, c := range n.Content {
		printNode(suf+"  ", c)
	}
}

type nodeType int

const (
	nodeTypeScl nodeType = iota
	nodeTypeSeq
	nodeTypeMap
)

type node struct {
	parent, key string
	value       interface{}
	line, colm  int
	typ         nodeType
}

// Parse parses a yaml tree from reader
func Parse(r io.Reader) (*yaml.Node, error) {
	d1 := yaml.NewDecoder(r)
	d1.KnownFields(true)
	var f yaml.Node
	if err := d1.Decode(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

func buildTree(n *yaml.Node) *node {
	r := node{
		line: n.Line,
		colm: n.Column,
	}

	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) > 0 {
			return buildTree(n.Content[0])
		}

	case yaml.MappingNode:
		tmp := map[string]yaml.Node{}
		n.Decode(&tmp)
		vn := map[string]*node{}
		for k, v := range tmp {
			vn[k] = buildTree(&v)
		}
		r.typ = nodeTypeMap
		r.value = vn

	case yaml.SequenceNode:
		tmp := []yaml.Node{}
		n.Decode(&tmp)
		vn := []*node{}
		for _, v := range tmp {
			vn = append(vn, buildTree(&v))
		}
		r.typ = nodeTypeSeq
		r.value = vn

	case yaml.ScalarNode:
		r.typ = nodeTypeScl
		n.Decode(&r.value)

	case yaml.AliasNode:
		v := buildTree(n.Alias)
		r.value = v.value
		r.typ = v.typ
	}

	return &r
}

func printMyNode(suf string, n *node) {
	switch v := n.value.(type) {
	case *node:
		printMyNode("  ", v)

	case []*node:
		fmt.Print("[")
		for _, t := range v {
			printMyNode("", t)
			fmt.Print(",")
		}
		fmt.Print("]")

	case map[string]*node:
		fmt.Print("{")
		for k, t := range v {
			fmt.Print(k, ":")
			printMyNode("", t)
			fmt.Print(",")
		}
		fmt.Print("}")

	default:
		fmt.Print(v)
	}
}

func diff(a, b *node) (add, sub []*node) {
	switch {
	case a == nil && b == nil:
	case a == nil:
		add = []*node{b}
	case b == nil:
		sub = []*node{a}
	case a.typ != b.typ:
		add = []*node{b}
		sub = []*node{a}
	default:
		goto inside
	}
	return

inside:
	switch a.typ {
	case nodeTypeScl:
		if a.value != b.value {
			add = []*node{b}
			sub = []*node{a}
		}

	case nodeTypeSeq:
		va := a.value.([]*node)
		vb := b.value.([]*node)

		l := len(va)
		if len(vb) < l {
			l = len(vb)
		}
		for i := 0; i < l; i++ {
			ad, su := diff(va[i], vb[i])
			add = append(add, ad...)
			sub = append(sub, su...)
		}
		for i := l; i < len(vb); i++ {
			add = append(add, vb[i])
		}
		for i := l; i < len(va); i++ {
			sub = append(sub, va[i])
		}

	case nodeTypeMap:
		va := a.value.(map[string]*node)
		vb := b.value.(map[string]*node)
		checked := map[string]bool{}
		for k := range va {
			ad, su := diff(va[k], vb[k])
			add = append(add, ad...)
			sub = append(sub, su...)
			checked[k] = true
		}
		for k := range vb {
			if checked[k] {
				continue
			}
			ad, su := diff(va[k], vb[k])
			add = append(add, ad...)
			sub = append(sub, su...)
			checked[k] = true
		}
	}

	return
}

// Diff returns the diffs of the files
func Diff(file1, file2 io.Reader) (Results, error) {
	f1, err := Parse(file1)
	if err != nil {
		return nil, err
	}

	f2, err := Parse(file2)
	if err != nil {
		return nil, err
	}

	add, sub := diff(buildTree(f1), buildTree(f2))
	for _, v := range sub {
		printMyNode("-", v)
		fmt.Println("\n--------------")
	}
	fmt.Println("==============")
	for _, v := range add {
		printMyNode("+", v)
		fmt.Println("\n--------------")
	}
	return nil, nil
}
