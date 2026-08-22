package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrInvalidFile  = errors.New("invalid file")
	ErrFileTooLarge = errors.New("file too large")
)

type Local struct {
	root    string
	maximum int64
	allowed map[string]map[string]bool
}

func NewLocal(root string, maximum int64) (*Local, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if maximum < 1 {
		return nil, ErrInvalidFile
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &Local{root: abs, maximum: maximum, allowed: map[string]map[string]bool{".csv": {"text/csv": true}, ".json": {"application/json": true}, ".pdf": {"application/pdf": true}, ".jpg": {"image/jpeg": true}, ".png": {"image/png": true}}}, nil
}

func (l *Local) Put(ctx context.Context, name, mime string, payload []byte) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if int64(len(payload)) > l.maximum {
		return "", "", ErrFileTooLarge
	}
	extension := strings.ToLower(filepath.Ext(name))
	if !l.allowed[extension][mime] {
		return "", "", ErrInvalidFile
	}
	detected := http.DetectContentType(payload)
	if !contentMatches(extension, mime, detected, payload) {
		return "", "", ErrInvalidFile
	}
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])
	fileID := checksum[:24] + extension
	if !regexp.MustCompile(`^[a-f0-9]{24}\.[a-z0-9]+$`).MatchString(fileID) {
		return "", "", ErrInvalidFile
	}
	destination := filepath.Join(l.root, fileID)
	if !strings.HasPrefix(destination, l.root+string(filepath.Separator)) {
		return "", "", ErrInvalidFile
	}
	temporary := destination + ".tmp"
	if err := os.WriteFile(temporary, payload, 0o640); err != nil {
		return "", "", err
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return "", "", err
	}
	return fileID, checksum, nil
}

func (l *Local) Open(ctx context.Context, fileID string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if filepath.Base(fileID) != fileID {
		return nil, ErrInvalidFile
	}
	return os.Open(filepath.Join(l.root, fileID))
}

func contentMatches(extension, declared, detected string, payload []byte) bool {
	if declared == detected {
		return true
	}
	if extension == ".csv" && declared == "text/csv" && strings.HasPrefix(detected, "text/plain") {
		return true
	}
	if extension == ".json" && declared == "application/json" && strings.HasPrefix(detected, "text/plain") {
		return json.Valid(payload)
	}
	return false
}
