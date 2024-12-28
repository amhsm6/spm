package main

import (
	"context"
	"fmt"
	"os"

	"spm/pkg/filetree"
	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v3"
)

func main() {
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
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Printf("Reading tree from %s\n", pkg)

			file, err := os.Open(pkg)
			if err != nil {
				return err
			}

			tree, err := filetree.Decode(file)
			if err != nil {
				return err
			}

			fmt.Println(tree)

			dst := c.String("dest")
			tree.Copy(dst)

			fmt.Printf("Transferred tree to %s\n", dst)

			return nil
		},
	}

	cmd := &cli.Command{
		Name:            "spm",
		UsageText:       "spm <COMMAND>",
		HideHelpCommand: true,
		Commands:        []*cli.Command{buildCmd, installCmd},
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
