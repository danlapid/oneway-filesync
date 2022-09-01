package zip

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yeka/zip"
)

const ZIPPASSWORD = "filesync"

func ZipFile(dst io.Writer, src *os.File) error {
	ziparchive := zip.NewWriter(dst)
	zipfile, err := ziparchive.Encrypt(filepath.Base(src.Name()), ZIPPASSWORD, zip.StandardEncryption)
	if err != nil {
		return fmt.Errorf("error creating file in zip: %v", err)
	}

	if _, err = io.Copy(zipfile, src); err != nil {
		return fmt.Errorf("error copying file contents: %v", err)
	}

	if err = ziparchive.Close(); err != nil {
		return fmt.Errorf("error closing zip file: %v", err)
	}
	return nil
}
