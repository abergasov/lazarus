package artifact_inspector

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"lazarus/internal/entities"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func (s *Service) scanTmpFile(ctx context.Context, tmp *os.File) error {
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind temp file: %w", err)
	}

	if err := s.avClient.ScanReader(ctx, tmp); err != nil {
		return fmt.Errorf("av scan: %w", err)
	}
	return nil
}

func (s *Service) detectMimeType(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind file: %w", err)
	}

	buf := make([]byte, sniffLen)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read sniff bytes: %w", err)
	}

	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind file: %w", err)
	}

	return http.DetectContentType(buf[:n]), nil
}

func classifyArtifact(mimeType string) entities.ArtifactClass {
	m := strings.ToLower(stripMimeParams(mimeType))

	switch {
	case strings.HasPrefix(m, "image/"):
		return entities.ArtifactClassImage
	case m == "application/pdf":
		return entities.ArtifactClassPDF
	case strings.HasPrefix(m, "text/"):
		return entities.ArtifactClassText
	case m == "application/json",
		m == "application/xml",
		m == "application/rtf",
		m == "application/msword",
		m == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return entities.ArtifactClassText
	default:
		return entities.ArtifactClassUnknown
	}
}

func stripMimeParams(s string) string {
	if i := strings.IndexByte(s, ';'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

func sameMimeFamily(detected, stored, declared string) bool {
	d := stripMimeParams(strings.ToLower(detected))
	s := stripMimeParams(strings.ToLower(stored))
	dec := stripMimeParams(strings.ToLower(declared))

	if d == s {
		return true
	}
	if s == "" && d == dec {
		return true
	}

	// docx/xlsx often look like zip for simple detectors
	if s == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" && d == "application/zip" {
		return true
	}
	if dec == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" && d == "application/zip" {
		return true
	}

	return false
}

func (s *Service) renderPDFPages(ctx context.Context, artifact *entities.Artifact, pdfPath string) error {
	outDir, err := os.MkdirTemp("", "pdf-pages-*")
	if err != nil {
		return fmt.Errorf("create temp page dir: %w", err)
	}
	defer os.RemoveAll(outDir) //nolint:errcheck

	prefix := filepath.Join(outDir, "page")

	cmd := exec.CommandContext(ctx, "pdftoppm", "-png", "-r", "150", pdfPath, prefix)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pdftoppm failed: %w: %s", err, string(out))
	}

	files, err := filepath.Glob(filepath.Join(outDir, "page-*.png"))
	if err != nil {
		return fmt.Errorf("glob rendered pages: %w", err)
	}
	sort.Strings(files)

	for i, file := range files {
		if err = s.uploadPDFPageImage(ctx, artifact, i+1, file); err != nil {
			return fmt.Errorf("upload page %d: %w", i+1, err)
		}
	}
	return nil
}

func (s *Service) uploadPDFPageImage(ctx context.Context, artifact *entities.Artifact, pageNum int, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open page image: %w", err)
	}
	defer f.Close() //nolint:errcheck

	h := sha256.New()
	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat page image: %w", err)
	}

	buf, err := io.ReadAll(io.TeeReader(f, h))
	if err != nil {
		return fmt.Errorf("read page image: %w", err)
	}

	key := fmt.Sprintf("%s/pages/%03d.png", artifact.ObjectKey, pageNum)
	if err = s.bucketClient.Upload(ctx, key, bytes.NewReader(buf), stat.Size()); err != nil {
		return fmt.Errorf("upload page image: %w", err)
	}

	if err = s.repo.CreateArtifactDerivative(ctx, &entities.ArtifactDerivatives{
		ArtifactID:   artifact.ID,
		Kind:         entities.ArtifactDerivativeTypePDFImagePage,
		PageNum:      sql.NullInt32{Int32: int32(pageNum), Valid: true},
		Storage:      artifact.Storage,
		Bucket:       artifact.Bucket,
		ObjectKey:    key,
		DetectedMIME: "image/png",
		ByteSize:     stat.Size(),
		SHA256Hex:    hex.EncodeToString(h.Sum(nil)),
	}); err != nil {
		return fmt.Errorf("create derivative row: %w", err)
	}
	return nil
}
