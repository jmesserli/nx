package cache

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"text/template"
)

var logger = log.New(os.Stdout, "[cached_writer] ", log.LstdFlags)

type CachedTemplateWriter struct {
	template        *template.Template
	ignorePatterns  []*regexp.Regexp
	useTabbedWriter bool
	newHashes       map[string]string
	ProcessedFiles  []string
	UpdatedFiles    []string
}

func New(template *template.Template, ignorePatterns []*regexp.Regexp, useTabbedWriter bool) *CachedTemplateWriter {
	return &CachedTemplateWriter{
		template:        template,
		ignorePatterns:  ignorePatterns,
		useTabbedWriter: useTabbedWriter,
		newHashes:       map[string]string{},
	}
}

func (cw *CachedTemplateWriter) WriteTemplate(
	file string,
	data interface{},
) (bool, error) {
	buf := bytes.Buffer{}
	err := func() error {
		var bufWriter io.Writer
		if cw.useTabbedWriter {
			tw := tabwriter.NewWriter(&buf, 2, 2, 2, ' ', 0)
			bufWriter = tw
			defer tw.Flush()
		} else {
			bufWriter = &buf
		}

		err := cw.template.Execute(bufWriter, data)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		return false, err
	}

	str := string(buf.Bytes())
	hashStr := cw.hash(str)

	existingFileStr, err := cw.getFileContent(file)
	if err == nil {
		existingHash := cw.hash(existingFileStr)
		if existingHash == hashStr {
			//logger.Printf("File fresh: %s\n", file)
			cw.ProcessedFiles = append(cw.ProcessedFiles, file)
			cw.newHashes[file] = existingHash
			return false, nil
		}
	} else {
		logger.Printf("ignored error while reading existing file %s: %s\n", file, err.Error())
	}

	f, err := os.Create(file)
	if err != nil {
		return false, err
	}
	defer func(closeable io.Closer) {
		err := closeable.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	_, err = f.Write(buf.Bytes())
	if err != nil {
		return false, err
	}

	logger.Printf("New hash %s for file %s\n", hashStr, file)
	cw.ProcessedFiles = append(cw.ProcessedFiles, file)
	cw.UpdatedFiles = append(cw.UpdatedFiles, file)
	cw.newHashes[file] = hashStr

	return true, nil
}

func (cw *CachedTemplateWriter) getFileContent(file string) (string, error) {
	stat, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if stat.IsDir() {
		return "", fmt.Errorf("%s is a directory", file)
	}

	fileBytes, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(fileBytes), nil
}

func (cw *CachedTemplateWriter) hash(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSpace(content)

	if cw.ignorePatterns != nil {
		for _, regex := range cw.ignorePatterns {
			content = regex.ReplaceAllString(content, "-hash:omit-")
		}
	}

	hash := sha1.New()
	hash.Write([]byte(content))
	hashBytes := hash.Sum(nil)
	return fmt.Sprintf("%x", hashBytes)
}
