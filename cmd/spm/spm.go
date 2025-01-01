package main

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
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
	
	installCmd := &cobra.Command{
		Use:  "install",
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			pkgname := args[0]
			paths := args[1:]

			dest, err := cmd.Flags().GetString("dest")
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%s %s\n", color.CyanString("Install"), pkgname)

			tree, err := filetree.Build(paths)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(tree)

			lock := struct{ Dest string; Tree *filetree.Tree }{
				Dest: dest,
				Tree: tree,
			}

			lockfile, err := os.Create(filepath.Join(LOCKDIR, pkgname))
			if err != nil {
				log.Fatal(err)
			}
			defer lockfile.Close()

			enc := gob.NewEncoder(lockfile)
			err = enc.Encode(lock)
			if err != nil {
				log.Fatal(err)
			}

			err = tree.Copy(dest)
			if err != nil {
				log.Fatal(err)
			}

			color.Green("Success")
		},
	}

	flags := installCmd.Flags()
	flags.StringP("dest", "d", "/home/anton/src/spm/test", "more usage")

	removeCmd := &cobra.Command{
		Use:  "remove",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pkgname := args[0]

			fmt.Printf("%s %s\n", color.RedString("Remove"), pkgname)

			lockpath := filepath.Join(LOCKDIR, pkgname)
			lockfile, err := os.Open(lockpath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Package is not installed")
					return
				} else {
					log.Fatal(err)
				}
			}

			lock := &struct{ Dest string; Tree *filetree.Tree }{}

			dec := gob.NewDecoder(lockfile)
			err = dec.Decode(lock)
			if err != nil {
				log.Fatal(err)
			}

			err = lockfile.Close()
			if err != nil {
				log.Fatal(err)
			}

			tree := lock.Tree
			dest := lock.Dest
			fmt.Println(tree)

			err = tree.Remove(dest)
			if err != nil {
				log.Fatal(err)
			}

			err = os.Remove(lockpath)
			if err != nil {
				log.Fatal(err)
			}

			color.Green("Success")
		},
	}

	rootCmd := &cobra.Command{
		Use: "spm",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	rootCmd.AddCommand(installCmd, removeCmd)

	rootCmd.Execute()
}
