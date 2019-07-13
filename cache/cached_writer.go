package cache

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"text/tabwriter"
	"text/template"
)

var logger = log.New(os.Stdout, "[cached_writer] ", log.LstdFlags)

type CachedTemplateWriter struct {
	hashFile   string
	fileHashes map[string]string
	buf        bytes.Buffer
}

func New(hashFile string) *CachedTemplateWriter {
	if _, err := os.Stat(hashFile); os.IsNotExist(err) {
		return &CachedTemplateWriter{
			hashFile:   hashFile,
			fileHashes: map[string]string{},
		}
	}

	jsonBytes, err := ioutil.ReadFile(hashFile)
	if err != nil {
		panic(err)
	}

	data := map[string]string{}
	err = json.Unmarshal(jsonBytes, &data)
	if err != nil {
		panic(err)
	}

	return &CachedTemplateWriter{
		hashFile:   hashFile,
		fileHashes: data,
	}
}

func (w *CachedTemplateWriter) WriteTemplate(
	file string,
	tpl *template.Template,
	data interface{},
	ignorePatterns []*regexp.Regexp,
	useTabbedWriter bool,
) (bool, error) {
	// Reset buffer
	w.buf = bytes.Buffer{}

	err := tpl.Execute(&w.buf, data)
	if err != nil {
		return false, err
	}

	str := string(w.buf.Bytes())
	for _, regex := range ignorePatterns {
		str = regex.ReplaceAllString(str, "-hash:omit-")
	}

	hash := sha1.New()
	hash.Write([]byte(str))
	hashBytes := hash.Sum(nil)
	hashStr := fmt.Sprintf("%x", hashBytes)

	existingHash, ok := w.fileHashes[file]
	if ok && existingHash == hashStr {
		logger.Printf("File fresh: %s\n", file)
		return false, nil
	}

	f, err := os.Create(file)
	if err != nil {
		return false, err
	}
	defer f.Close()

	var writer io.Writer
	if useTabbedWriter {
		writer = tabwriter.NewWriter(f, 2, 2, 2, ' ', 0)
	} else {
		writer = f
	}

	if useTabbedWriter {
		wr := tabwriter.NewWriter(f, 2, 2, 2, ' ', 0)
		_, err = wr.Write(w.buf.Bytes())
		if err != nil {
			return false, err
		}
		_ = wr.Flush()
	} else {
		_, err = writer.Write(w.buf.Bytes())
		if err != nil {
			return false, err
		}
	}

	logger.Printf("New hash %s for file %s\n", hashStr, file)
	w.fileHashes[file] = hashStr
	w.updateJson()

	return true, nil
}

func (w *CachedTemplateWriter) updateJson() {
	jsonBytes, err := json.Marshal(w.fileHashes)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(w.hashFile)
	if err != nil {
		panic(err)
	}

	_, err = f.Write(jsonBytes)
	if err != nil {
		panic(err)
	}

	_ = f.Close()
}
