package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
)

const (
	errorExitOpen = iota + 10
	errorExitWrite
	errorExitReadDir
	errorExitReadSplit
	errorExitMakeDir
	errorExitMoveFile
	errorExitBadFile
	errorExitAddDir
	errorExitCwd
	errorExitEmptySplit
)

var CLI struct {
	Add   AddCmd   `cmd:"" help:"Mark split points by adding filenames to .split."`
	Split SplitCmd `cmd:"" help:"Split files using markers stored in .split."`
}

type AddCmd struct {
	Files []string `arg:"" help:"Filenames used as split markers."`
}

type SplitCmd struct{}

const splitFile = ".split"

func main() {
	ctx := kong.Parse(
		&CLI,
		kong.Description("Split files in a directory into sub directories using added files."),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

type errorWithCode struct {
	err  error
	code int
}

func (v *errorWithCode) Error() string {
	return v.err.Error()
}

func (v *errorWithCode) ExitCode() int {
	return v.code
}

func (v *AddCmd) Run() error {
	f, err := os.OpenFile(splitFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return &errorWithCode{
			err:  fmt.Errorf("failed to open %s - %w", splitFile, err),
			code: errorExitOpen,
		}
	}
	defer f.Close()

	for _, file := range v.Files {
		if stat, err := os.Stat(file); err != nil || stat.IsDir() {
			if err != nil {
				return &errorWithCode{
					err:  fmt.Errorf("file probably doesn't exist: %s - %w", file, err),
					code: errorExitBadFile,
				}
			}
			return &errorWithCode{
				err:  fmt.Errorf("can't add dirs: %s", file),
				code: errorExitAddDir,
			}
		}
		_, err := f.WriteString(file + "\n")
		if err != nil {
			return &errorWithCode{
				err:  fmt.Errorf("failed to add %s - %w", file, err),
				code: errorExitWrite,
			}
		}
	}
	return nil
}

func (*SplitCmd) Run() error {
	base, err := os.Getwd()
	if err != nil {
		return &errorWithCode{
			err:  fmt.Errorf("failed get cwd - %w", err),
			code: errorExitCwd,
		}
	}
	allFiles, err := readDirFiles(base)
	if err != nil {
		return &errorWithCode{
			err:  fmt.Errorf("failed to read dir files - %w", err),
			code: errorExitReadDir,
		}
	}
	sort.Strings(allFiles)

	markers, err := readLines(splitFile)
	if err != nil {
		return &errorWithCode{
			err:  fmt.Errorf("failed to read split file lines - %w", err),
			code: errorExitReadSplit,
		}
	}
	if len(markers) == 0 {
		return &errorWithCode{
			err:  errors.New("empty split file"),
			code: errorExitEmptySplit,
		}
	}
	sort.Strings(markers)

	partitions := partitionFiles(allFiles, markers)
	for i, files := range partitions {
		outDir := fmt.Sprintf("split_%d", i)
		if err := os.MkdirAll(outDir, 0700); err != nil {
			return &errorWithCode{
				err:  fmt.Errorf("failed to create directory %s - %w", outDir, err),
				code: errorExitMakeDir,
			}
		}
		for _, file := range files {
			dst := filepath.Join(outDir, file)
			if err := os.Rename(file, dst); err != nil {
				return &errorWithCode{
					err:  fmt.Errorf("failed move %s - %w", file, err),
					code: errorExitMoveFile,
				}
			}
		}
	}
	return os.Remove(splitFile)
}

func readDirFiles(dir string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Name() == splitFile {
			continue
		}
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

func readLines(path string) ([]string, error) {
	var lines []string
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

func partitionFiles(all, markers []string) [][]string {
	var partitions [][]string
	markerSet := make(map[string]bool)
	for _, m := range markers {
		markerSet[m] = true
	}

	var current []string
	for _, f := range all {
		if markerSet[f] {
			if len(current) > 0 {
				partitions = append(partitions, current)
			}
			current = []string{f}
		} else {
			current = append(current, f)
		}
	}
	if len(current) > 0 {
		partitions = append(partitions, current)
	}
	return partitions
}
