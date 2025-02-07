package util

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

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

func WaitForKey() error {
    err := exec.Command("stty", "-F", "/dev/tty", "cbreak").Run()
    if err != nil {
        return err
    }

    fmt.Print("Press any key to continue... ")

    buf := make([]byte, 1)
    _, err = os.Stdin.Read(buf)
    if err != nil {
        return err
    }

    err = exec.Command("stty", "-F", "/dev/tty", "-cbreak").Run()
    if err != nil {
        return err
    }

    fmt.Println()
    return nil
}
