package main

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"
	"spm/pkg/filetree"
)

const LOCKDIR = "/home/anton/src/spm/var/lib/spm"

func main() {
	gob.Register(filetree.NodeFile{})
	gob.Register(filetree.NodeDir{})
	gob.Register(filetree.NodeSymLink{})

	err := os.MkdirAll(LOCKDIR, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	
	var paths []string
	buildCmd := &cli.Command{
		Name:      "build",
		Aliases:   []string{"b"},
		UsageText: "spm build [OPTIONS] PATH...",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:   "PATH",
				Min:    1,
				Max:    -1,
				Values: &paths,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "package.spk",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			tree, err := filetree.Build(paths)
			if err != nil {
				return err
			}

			fmt.Println(tree)

			output := c.String("output")
			file, err := os.Create(output)
			if err != nil {
				return err
			}

			defer file.Close()

			err = tree.Encode(file)
			if err != nil {
				return err
			}

			fmt.Printf("Wrote tree to %s\n", output)

			return nil
		},
	}

	var pkg string
	installCmd := &cli.Command{
		Name:      "install",
		Aliases:   []string{"i"},
		UsageText: "spm install [OPTIONS] PACKAGE",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "PACKAGE",
				Min:         1,
				Max:         1,
				Destination: &pkg,
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dest",
				Aliases: []string{"d"},
				Value:   "/home/anton/src/spm/test",
			},
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			dst := c.String("dest")

			lockname, ok := strings.CutSuffix(pkg, filepath.Ext(pkg))
			if !ok {
				return errors.New("impossible")
			}

			if c.IsSet("name") {
				lockname = c.String("name")
			}

			fmt.Printf("Reading tree from %s\n", pkg)

			file, err := os.Open(pkg)
			if err != nil {
				return err
			}

			defer file.Close()

			tree, err := filetree.Decode(file)
			if err != nil {
				return err
			}

			fmt.Println(tree)

			lock := struct{ Dest string; Tree *filetree.Tree }{
				Dest: dst,
				Tree: tree,
			}

			lockfile, err := os.Create(filepath.Join(LOCKDIR, lockname))
			if err != nil {
				return err
			}

			defer lockfile.Close()

			enc := gob.NewEncoder(lockfile)
			err = enc.Encode(lock)
			if err != nil {
				return err
			}

			err = tree.Copy(dst)
			if err != nil {
				return err
			}

			fmt.Printf("Transferred tree to %s\n", dst)

			return nil
		},
	}

	var lockname string
	removeCmd := &cli.Command{
		Name:      "remove",
		Aliases:   []string{"r"},
		UsageText: "spm remove [OPTIONS] PACKAGE",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "PACKAGE",
				Min:         1,
				Max:         1,
				Destination: &lockname,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			lockpath := filepath.Join(LOCKDIR, lockname)

			lockfile, err := os.Open(lockpath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Package is not installed")
				}

				return err
			}
			
			defer lockfile.Close()

			lock := &struct{ Dest string; Tree *filetree.Tree }{}

			dec := gob.NewDecoder(lockfile)
			err = dec.Decode(lock)
			if err != nil {
				return err
			}

			fmt.Println(lock.Tree)

			err = lock.Tree.Remove(lock.Dest)
			if err != nil {
				return err
			}

			fmt.Printf("Removed tree from %s\n", lock.Dest)

			err = os.Remove(lockpath)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd := &cli.Command{
		Name:            "spm",
		UsageText:       "spm <COMMAND>",
		HideHelpCommand: true,
		Commands:        []*cli.Command{buildCmd, installCmd, removeCmd},
	}

	err = cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
