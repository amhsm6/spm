package filetree

import (
	"io"
	"encoding/gob"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

type Tree struct {
	Name     string
	Node     Node
	Children map[string]*Tree
}

type Node interface{}

type NodeFile struct {
	Data []byte
}

type NodeDir struct{}

type NodeSymLink struct {
	Target string
}

func Build(paths []string) (*Tree, error) {
	trees := make(map[string]*Tree)
	for _, path := range paths {
		tree, err := buildTree(path)
		if err != nil {
			return nil, err
		}

		if filepath.Clean(tree.Name) == "." || filepath.Clean(tree.Name) == ".." {
			maps.Copy(trees, tree.Children)
		} else {
			trees[tree.Name] = tree
		}
	}

	root := &Tree{Name: "/", Node: NodeDir{}, Children: trees}
	return root, nil
}

func buildTree(path string) (*Tree, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	rootname := filepath.Base(path) + string(os.PathSeparator)
	tree := &Tree{
		Name: rootname,
		Node: NodeDir{},
		Children: make(map[string]*Tree),
	}

	for _, entry := range entries {
		entryname := entry.Name()
		entrypath := filepath.Join(path, entryname)

		switch entry.Type() {
		case os.ModeDir:
			subtree, err := buildTree(entrypath)
			if err != nil {
				return nil, err
			}

			dirname := entryname + string(os.PathSeparator)
			tree.Children[dirname] = subtree

		case os.ModeSymlink:
			return nil, errors.New("symlinks not implemented")

		default:
			data, err := os.ReadFile(entrypath)
			if err != nil {
				return nil, err
			}

			node := NodeFile{data}
			tree.Children[entryname] = &Tree{Name: entryname, Node: node}
		}
	}

	return tree, nil
}

func (t *Tree) Encode(w io.Writer) error {
	gob.Register(NodeFile{})
	gob.Register(NodeDir{})
	gob.Register(NodeSymLink{})

	enc := gob.NewEncoder(w)
	return enc.Encode(t)
}

func Decode(r io.Reader) (*Tree, error) {
	tree := &Tree{}

	gob.Register(NodeFile{})
	gob.Register(NodeDir{})
	gob.Register(NodeSymLink{})

	dec := gob.NewDecoder(r)
	err := dec.Decode(tree)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func (t *Tree) Copy(dst string) error {
	for _, subtree := range t.Children {
		path := filepath.Join(dst, subtree.Name)

		switch node := subtree.Node.(type) {
		case NodeDir:
			err := os.Mkdir(path, os.ModePerm)
			if err != nil {
				return err
			}

			subtree.Copy(path)
		
		case NodeFile:
			file, err := os.Create(path)
			if err != nil {
				return err
			}

			_, err = file.Write(node.Data)
			if err != nil {
				return err
			}

		case NodeSymLink:
			return errors.New("symlinks not implemented")

		default:
			return errors.New("Impossible")
		}
	}

	return nil
}

func (t *Tree) String() string {
	return t.render([]bool{})
}

func (t *Tree) render(bars []bool) string {
	var out strings.Builder

	if _, ok := t.Node.(NodeDir); ok {
		out.WriteString("\u001b[38;2;254;40;162m" + t.Name + "\u001b[0m\n")
	} else if node, ok := t.Node.(NodeFile); ok {
		bytes := len(node.Data)

		size := fmt.Sprintf("%d", bytes)
		if bytes > 1024 * 1024 {
			size = fmt.Sprintf("%.2f MB", float32(bytes) / (1024 * 1024))
		} else if bytes > 1024 {
			size = fmt.Sprintf("%.2f KB", float32(bytes) / 1024)
		}

		out.WriteString(fmt.Sprintf("%s \u001b[32m{%s}\u001b[0m\n", t.Name, size))
	}

	index := 0
	for _, subtree := range t.Children {
		for _, b := range bars { 
			if b {
				out.WriteString("│")
			} else {
				out.WriteString(" ")
			}
			out.WriteString(strings.Repeat(" ", 3))
		}

		if index == len(t.Children) - 1 {
			out.WriteString("└── ")
			out.WriteString(subtree.render(append(bars, false)))
		} else {
			out.WriteString("├── ")
			out.WriteString(subtree.render(append(bars, true)))
		}

		index++
	}

	return out.String()
}
