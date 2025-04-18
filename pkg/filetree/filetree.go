package filetree

import (
    "encoding/gob"
    "errors"
    "fmt"
    "io"
    "maps"
    "os"
    "path/filepath"
    "strings"

    "github.com/fatih/color"
    "spm/pkg/util"
)

type Tree struct {
    Name     string
    Node     Node
    Children map[string]*Tree
}

type Node any

type NodeFile struct {
    Data        []byte
    Permissions os.FileMode
}

type NodeDir struct{}

type NodeSymLink struct {
    Target string
}

func Build(paths []string, prefix string) (*Tree, error) {
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

    if !filepath.IsAbs(prefix) {
        return nil, errors.New("prefix must be absolute")
    }

    for {
        dir, file := filepath.Split(filepath.Clean(prefix))
        if filepath.Ext(file) != "" {
            return nil, errors.New("prefix must contain only directories")
        }

        if file == "" {
            break
        }

        dirname := file + string(os.PathSeparator)
        tree := &Tree{
            Name:     dirname,
            Node:     NodeDir{},
            Children: trees,
        }

        trees = make(map[string]*Tree)
        trees[dirname] = tree
        
        prefix = dir
    }

    root := &Tree{ Name: "/", Node: NodeDir{}, Children: trees }
    return root, nil
}

func buildTree(path string) (*Tree, error) {
	stat, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		name := filepath.Base(path)

        if stat.Mode().IsRegular() {
            data, err := os.ReadFile(path)
            if err != nil {
                return nil, err
            }

            node := NodeFile{ Data: data, Permissions: stat.Mode() }
            return &Tree{ Name: name, Node: node }, nil
        }

        switch stat.Mode() {
        case os.ModeSymlink:
            target, err := os.Readlink(path)
            if err != nil {
                return nil, err
            }

            if filepath.IsAbs(target) {
                abspath, err := filepath.Abs(path)
                if err != nil {
                    return nil, err
                }

                target, err = filepath.Rel(abspath, target)
                if err != nil {
                    return nil, err
                }
            }

            node := NodeSymLink{ Target: target }
            return &Tree{ Name: name, Node: node }, nil

        default:
            return nil, fmt.Errorf("file mode %v of %v unsupported", stat.Mode(), path)
        }
	}

    entries, err := os.ReadDir(path)
    if err != nil {
        return nil, err
    }

    rootname := filepath.Base(path) + string(os.PathSeparator)
    tree := &Tree{
        Name:     rootname,
        Node:     NodeDir{},
        Children: make(map[string]*Tree),
    }

    for _, entry := range entries {
        entryname := entry.Name()
        entrypath := filepath.Join(path, entryname)

        if entry.IsDir() {
            subtree, err := buildTree(entrypath)
            if err != nil {
                return nil, err
            }

            dirname := entryname + string(os.PathSeparator)
            tree.Children[dirname] = subtree

            continue
        }

        if entry.Type().IsRegular() {
            data, err := os.ReadFile(entrypath)
            if err != nil {
                return nil, err
            }

            info, err := os.Lstat(entrypath)
            if err != nil {
                return nil, err
            }

            node := NodeFile{ Data: data, Permissions: info.Mode() }
            tree.Children[entryname] = &Tree{ Name: entryname, Node: node }

            continue
        }

        switch entry.Type() {
        case os.ModeSymlink:
            target, err := os.Readlink(entrypath)
            if err != nil {
                return nil, err
            }

            if filepath.IsAbs(target) {
                abspath, err := filepath.Abs(path)
                if err != nil {
                    return nil, err
                }

                target, err = filepath.Rel(abspath, target)
                if err != nil {
                    return nil, err
                }
            }

            node := NodeSymLink{ Target: target }
            tree.Children[entryname] = &Tree{ Name: entryname, Node: node }

        default:
            return nil, fmt.Errorf("file mode %v of %v unsupported", entry.Type(), entrypath)
        }
    }

    return tree, nil
}

func (t *Tree) Encode(w io.Writer) error {
    enc := gob.NewEncoder(w)
    return enc.Encode(t)
}

func Decode(r io.Reader) (*Tree, error) {
    tree := &Tree{}

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
            if !util.Exists(path) {
                err := os.Mkdir(path, os.ModePerm)
                if err != nil {
                    return err
                }
            }

			err := subtree.Copy(path)
			if err != nil {
				return err
			}
        
        case NodeFile:
            file, err := os.Create(path)
            if err != nil {
                return err
            }
            defer file.Close()

            _, err = file.Write(node.Data)
            if err != nil {
                return err
            }

            err = file.Chmod(node.Permissions)
            if err != nil {
                return err
            }

        case NodeSymLink:
            err := os.Symlink(node.Target, path)
            if err != nil {
                return err
            }

        default:
            return errors.New("impossible")
        }
    }

    return nil
}

func (t *Tree) Remove(dst string) error {
    for _, subtree := range t.Children {
        path := filepath.Join(dst, subtree.Name)

        switch subtree.Node.(type) {
        case NodeDir:
            err := subtree.Remove(path)
            if err != nil {
                return err
            }

            if util.Empty(path) {
                err = os.Remove(path)
                if err != nil {
                    return err
                }
            }
        
        case NodeFile, NodeSymLink:
            if util.Exists(path) {
                err := os.Remove(path)
                if err != nil {
                    return err
                }
            }

        default:
            return errors.New("impossible")
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
        color.RGB(254, 40, 162).Fprintln(&out, t.Name)
    } else if node, ok := t.Node.(NodeFile); ok {
        bytes := len(node.Data)

        size := fmt.Sprintf("%d", bytes)
        if bytes > 1024*1024 {
            size = fmt.Sprintf("%.2f MB", float32(bytes) / (1024*1024))
        } else if bytes > 1024 {
            size = fmt.Sprintf("%.2f KB", float32(bytes) / 1024)
        }

        out.WriteString(t.Name + color.GreenString(" {%s}\n", size))
    } else if node, ok := t.Node.(NodeSymLink); ok {
        out.WriteString(t.Name + color.CyanString(" {%s}\n", node.Target))
    } else {
        panic("impossible")
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
