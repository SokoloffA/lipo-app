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
	inPaths []string
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

func newEngine(in []string, out string) (Engine, error) {
	res := Engine{
		inPaths: in,
		outPath: out,
		verbose: false,
	}
	var err error

	for i, p := range in {
		if res.inPaths[i], err = filepath.Abs(p); err != nil {
			return res, err
		}
	}

	if res.outPath, err = filepath.Abs(out); err != nil {
		return res, err
	}

	return res, nil
}

func (e Engine) run() error {
	if e.verbose {
		for i, p := range e.inPaths {
			fmt.Printf("Input %d: %s", i, p)
		}
		fmt.Println("Output:  ", e.outPath)
	}

	for i, p1 := range e.inPaths {

		if p1 == e.outPath {
			return fmt.Errorf("the name of the output bundle must not match the input")
		}

		if !dirExists(p1 + "/Contents/MacOS") {
			return fmt.Errorf("input bundle %s not found", p1)
		}

		for _, p2 := range e.inPaths[i+1:] {
			if p1 == p2 {
				return fmt.Errorf("the same paths are set for the input bundles")
			}
		}
	}

	os.RemoveAll(e.outPath)

	err := filepath.Walk(e.inPaths[0],
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
	files := make([]string, len(e.inPaths))
	for i, in := range e.inPaths {
		files[i] = strings.Replace(path, e.inPaths[0], in, 1)
	}

	out := strings.Replace(path, e.inPaths[0], e.outPath, 1)

	if fileInfo.IsDir() {
		return e.processDir(out, fileInfo)
	}

	if fileInfo.Mode().Type()&os.ModeSymlink != 0 {
		return e.processSymlink(files, out)
	}

	if fileInfo.Mode().IsRegular() {
		return e.processFile(files, out, fileInfo)
	}

	// return fmt.Errorf("unsupported file type %s", in1)
	return nil
}

func (e Engine) processDir(out string, fileInfo os.FileInfo) error {
	if e.verbose {
		fmt.Println("  - create directory")
		fmt.Println("     - out: ", out)
	}
	return os.MkdirAll(out, fileInfo.Mode())
}

func (e Engine) processSymlink(files []string, out string) error {
	dest, err := os.Readlink(files[0])
	if err != nil {
		return err
	}
	dest = strings.Replace(dest, e.inPaths[0], e.outPath, 1)

	if e.verbose {
		fmt.Println("  - create symlink")
		fmt.Println("     - out:     ", out)
		fmt.Println("     - link to: ", dest)
	}

	return os.Symlink(dest, out)
}

func (e Engine) processFile(files []string, out string, fileInfo os.FileInfo) error {
	if len(files) == 1 {
		return e.copyFile(files[0], out)
	}

	f, err := os.Open(files[0])
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
		return e.processBinFile(files, out, fileInfo)
	}

	return e.copyFile(files[0], out)
}

func (e Engine) processBinFile(files []string, out string, fileInfo os.FileInfo) error {
	if e.verbose {
		fmt.Println("  - create universal binary")
		fmt.Println("     - out:   ", out)
		for i, f := range files {
			fmt.Printf("     - src %d: %s\n", i, f)
		}
	}

	args := []string{"-create", "-output", out}
	args = append(args, files...)

	cmd := exec.Command("lipo", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func (e Engine) copyFile(in, out string) error {
	if e.verbose {
		fmt.Println("  - copy file")
		fmt.Println("     - out: ", out)
		fmt.Println("     - src: ", in)
	}

	return copyFile(in, out)
}
