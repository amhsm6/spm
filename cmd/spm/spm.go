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

const LOCKDIR = "/var/lib/spm"

func main() {
	gob.Register(filetree.NodeFile{})
	gob.Register(filetree.NodeDir{})
	gob.Register(filetree.NodeSymLink{})

	err := os.MkdirAll(LOCKDIR, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	
	installCmd := &cobra.Command{
		Use:  "install PACKAGE_NAME PATHS...",
		Args: cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			pkgname := args[0]
			paths := args[1:]

			lockpath := filepath.Join(LOCKDIR, pkgname)
			_, err := os.Stat(lockpath)
			if !os.IsNotExist(err) {
				if err == nil {
					fmt.Printf("%s Package already exists\n", color.RedString("ERROR"))
					return
				} else {
					log.Fatal(err)
				}
			}

			prefix, err := cmd.Flags().GetString("prefix")
			if err != nil {
				log.Fatal(err)
			}

			dest, err := cmd.Flags().GetString("dest")
			if err != nil {
				log.Fatal(err)
			}

			tree, err := filetree.Build(paths, prefix)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(tree)
			fmt.Printf("%s %s\n", color.CyanString("Installing"), pkgname)

			lock := struct{ Dest string; Tree *filetree.Tree }{
				Dest: dest,
				Tree: tree,
			}

			lockfile, err := os.Create(lockpath)
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
	flags.StringP("prefix", "p", "/", "Prefix of the built tree")
	flags.StringP("dest", "d", "/", "Destination path")

	removeCmd := &cobra.Command{
		Use:  "remove PACKAGE_NAME",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pkgname := args[0]

			lockpath := filepath.Join(LOCKDIR, pkgname)
			lockfile, err := os.Open(lockpath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("%s Package not found\n", color.RedString("ERROR"))
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
			fmt.Printf("%s %s\n", color.RedString("Removing"), pkgname)

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
