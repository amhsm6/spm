package util

import (
    "errors"
    "fmt"
    "io"
    "os"

    "github.com/fatih/color"
)

func Empty(dir string) bool {
    f, err := os.Open(dir)
    if err != nil {
        return false
    }
    defer f.Close()

    _, err = f.Readdir(1)
    return errors.Is(err, io.EOF)
}

func Exists(path string) bool {
    _, err := os.Lstat(path)
    return err == nil
}

func Error(a ...any) {
    fmt.Fprintln(os.Stderr, color.RedString("ERROR ") + fmt.Sprint(a...))
    os.Exit(1)
}
