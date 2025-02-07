package main

import (
    "encoding/gob"
    "fmt"
    "os"
    "path/filepath"

    "github.com/fatih/color"
    "github.com/spf13/cobra"
    "spm/pkg/filetree"
    "spm/pkg/util"
)

const LOCKDIR = "/var/lib/spm"

func main() {
    gob.Register(filetree.NodeFile{})
    gob.Register(filetree.NodeDir{})
    gob.Register(filetree.NodeSymLink{})

    err := os.MkdirAll(LOCKDIR, os.ModePerm)
    if err != nil {
        util.Error(err)
    }
    
    installCmd := &cobra.Command{
        Use:   "install PACKAGE_NAME PATHS...",
        Short: "Install a package",
        Args:  cobra.MinimumNArgs(2),
        Run: func(cmd *cobra.Command, args []string) {
            pkgname := args[0]
            paths   := args[1:]

            lockpath := filepath.Join(LOCKDIR, pkgname)
            if util.Exists(lockpath) {
                util.Error("Package already exists")
            }

            prefix, err := cmd.Flags().GetString("prefix")
            if err != nil {
                util.Error(err)
            }

            dest, err := cmd.Flags().GetString("dest")
            if err != nil {
                util.Error(err)
            }

            tree, err := filetree.Build(paths, prefix)
            if err != nil {
                util.Error(err)
            }

            fmt.Println(tree)
            err = util.WaitForKey()
            if err != nil {
                util.Error(err)
            }
            fmt.Printf("%s %s\n", color.CyanString("Installing"), pkgname)

            lock := struct{ Dest string; Tree *filetree.Tree }{
                Dest: dest,
                Tree: tree,
            }

            lockfile, err := os.Create(lockpath)
            if err != nil {
                util.Error(err)
            }
            defer lockfile.Close()

            enc := gob.NewEncoder(lockfile)
            err = enc.Encode(lock)
            if err != nil {
                util.Error(err)
            }

            err = tree.Copy(dest)
            if err != nil {
                util.Error(err)
            }

            color.Green("Success")
        },
    }

    flags := installCmd.Flags()
    flags.StringP("prefix", "p", "/", "Prefix of the built tree")
    flags.StringP("dest", "d", "/", "Destination path")

    removeCmd := &cobra.Command{
        Use:   "remove PACKAGE_NAME",
        Short: "Remove a package",
        Args:  cobra.ExactArgs(1),
        Run: func(cmd *cobra.Command, args []string) {
            pkgname := args[0]

            lockpath := filepath.Join(LOCKDIR, pkgname)
            if !util.Exists(lockpath) {
                util.Error("Package not found")
            }

            lockfile, err := os.Open(lockpath)
            if err != nil {
                util.Error(err)
            }

            lock := &struct{ Dest string; Tree *filetree.Tree }{}

            dec := gob.NewDecoder(lockfile)
            err = dec.Decode(lock)
            if err != nil {
                util.Error(err)
            }

            err = lockfile.Close()
            if err != nil {
                util.Error(err)
            }

            tree := lock.Tree
            dest := lock.Dest

            fmt.Println(tree)
            err = util.WaitForKey()
            if err != nil {
                util.Error(err)
            }
            fmt.Printf("%s %s\n", color.RedString("Removing"), pkgname)

            err = tree.Remove(dest)
            if err != nil {
                util.Error(err)
            }

            err = os.Remove(lockpath)
            if err != nil {
                util.Error(err)
            }

            color.Green("Success")
        },
    }

    rootCmd := &cobra.Command{
        Use:               "spm",
        CompletionOptions: cobra.CompletionOptions{ DisableDefaultCmd: true },
    }

    rootCmd.AddCommand(installCmd, removeCmd)

    rootCmd.Execute()
}
