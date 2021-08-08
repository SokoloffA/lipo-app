package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Engine struct {
	inPath1 string
	inPath2 string
	outPath string
	verbose bool
}

func dirExists(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return fi.IsDir()
}

// func fileExists(name string) bool {
// 	fi, err := os.Stat(name)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			return false
// 		}
// 	}
// 	return !fi.IsDir()
// }

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

func newEngine(in1, in2, out string) (Engine, error) {
	res := Engine{}
	var err error
	if res.inPath1, err = filepath.Abs(in1); err != nil {
		return res, err
	}

	if res.inPath2, err = filepath.Abs(in2); err != nil {
		return res, err
	}

	if res.outPath, err = filepath.Abs(out); err != nil {
		return res, err
	}

	return res, nil
}

func (e Engine) run() error {
	if e.verbose {
		fmt.Println("Input 1: ", e.inPath1)
		fmt.Println("Input 2: ", e.inPath2)
		fmt.Println("Output:  ", e.outPath)
	}
	if e.inPath1 == e.inPath2 {
		return fmt.Errorf("the same paths are set for the input bundles")
	}

	if e.inPath1 == e.outPath || e.inPath2 == e.outPath {
		return fmt.Errorf("the name of the output bundle must not match the input")
	}

	if !dirExists(e.inPath1 + "/Contents/MacOS") {
		return fmt.Errorf("input bundle %s not found", e.inPath1)
	}

	if !dirExists(e.inPath2 + "/Contents/MacOS") {
		return fmt.Errorf("input bundle %s not found", e.inPath2)
	}

	os.RemoveAll(e.outPath)

	err := filepath.Walk(e.inPath1,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return e.processPath(path, info)
		})

	if err != nil {
		return err
	}

	return nil
}

func (e Engine) processPath(path string, fileInfo os.FileInfo) error {
	in1 := path
	in2 := strings.Replace(path, e.inPath1, e.inPath2, 1)
	out := strings.Replace(path, e.inPath1, e.outPath, 1)

	if fileInfo.IsDir() {
		return e.processDir(in1, in2, out, fileInfo)
	}

	if fileInfo.Mode().Type()&os.ModeSymlink != 0 {
		return e.processSymlink(in1, in2, out, fileInfo)
	}

	if fileInfo.Mode().IsRegular() {
		return e.processFile(in1, in2, out, fileInfo)
	}

	return fmt.Errorf("unsupported file type %s", in1)
}

func (e Engine) processDir(in1, in2, out string, fileInfo os.FileInfo) error {
	if e.verbose {
		fmt.Println("  - create directory")
		fmt.Println("     - out: ", out)
	}
	return os.MkdirAll(out, fileInfo.Mode())
}

func (e Engine) processSymlink(in1, in2, out string, fileInfo os.FileInfo) error {
	dest, err := os.Readlink(in1)
	if err != nil {
		return err
	}
	dest = strings.Replace(dest, e.inPath1, e.outPath, 1)

	if e.verbose {
		fmt.Println("  - create symlink")
		fmt.Println("     - out:     ", out)
		fmt.Println("     - link to: ", dest)
	}

	return os.Symlink(dest, out)
}

func (e Engine) processFile(in1, in2, out string, fileInfo os.FileInfo) error {
	f, err := os.Open(in1)
	if err != nil {
		return err
	}
	defer f.Close()

	data := make([]byte, 4)
	_, err = f.Read(data)
	if err != nil {
		return err
	}

	maco64_Magic := [4]byte{0xfe, 0xed, 0xfa, 0xcf}  // the 64-bit mach magic number
	maco64_Magic2 := [4]byte{0xcf, 0xfa, 0xed, 0xfe} // NXSwapInt(MH_MAGIC_64)

	if bytes.Equal(data[:4], maco64_Magic[:]) || bytes.Equal(data[:4], maco64_Magic2[:]) {
		return e.processBinFile(in1, in2, out, fileInfo)
	}

	return e.copyFile(in1, in2, out, fileInfo)
}

func (e Engine) processBinFile(in1, in2, out string, fileInfo os.FileInfo) error {
	if e.verbose {
		fmt.Println("  - create universal binary")
		fmt.Println("     - out:   ", out)
		fmt.Println("     - src 1: ", in1)
		fmt.Println("     - src 2: ", in2)
	}

	cmd := exec.Command("lipo", "-create", "-output", out, in1, in2)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func (e Engine) copyFile(in1, in2, out string, fileInfo os.FileInfo) error {
	if e.verbose {
		fmt.Println("  - copy file")
		fmt.Println("     - out: ", out)
		fmt.Println("     - src: ", in1)
	}

	return copyFile(in1, out)
}
