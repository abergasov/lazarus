package artifact_manager_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"lazarus/internal/entities"
	"lazarus/internal/service/artifact_manager"
	testhelpers "lazarus/internal/test_helpers"
	"lazarus/internal/test_helpers/seed"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeName(t *testing.T) {
	table := map[string]string{
		"report.pdf":                           "report.pdf",
		"dir/subdir/file.txt":                  "dir_subdir_file.txt",
		`dir\subdir\file.txt`:                  "dir_subdir_file.txt",
		`dir/subdir\file.txt`:                  "dir_subdir_file.txt",
		"ab\x00cd\x1fef":                       "abcdef",
		"ab\x7fcd":                             "abcd",
		"":                                     "file",
		"CON":                                  "file_CON",
		`\x00\x01 . . `:                        "_x00_x01",
		"\x00\x01\x1f\x7f":                     "file",
		`/\//\\`:                               "______",
		"привет-мир.pdf":                       "привет-мир.pdf",
		"привет 世界.pdf":                        "привет 世界.pdf",
		"com1":                                 "file_com1",
		strings.Repeat("a", 130):               strings.Repeat("a", 120),
		strings.Repeat("界", 130):               strings.Repeat("界", 120),
		strings.Repeat("a", 119) + "/" + "zzz": strings.Repeat("a", 119) + "_",
	}

	for in, want := range table {
		require.Equal(t, want, artifact_manager.SafeName(in))
	}
}

type testCase struct {
	filename       string
	contentType    string
	payload        []byte
	maxBytes       int64
	repoErr        error
	wantErr        string
	wantUpload     bool
	wantDelete     bool
	wantSafeName   string
	wantDetectMIME string
}

func TestServiceUploadNegative(t *testing.T) {
	// given
	container := testhelpers.GetClean(t)
	user := seed.NewUserBuilder().PopulateTests(t, container)
	tests := map[string]testCase{
		"reject empty": {
			filename:    "empty.txt",
			contentType: "text/plain",
			payload:     []byte{},
			maxBytes:    1024,
			wantErr:     "empty file",
		},
		"reject too large": {
			filename:    "big.bin",
			contentType: "application/octet-stream",
			payload:     bytes.Repeat([]byte("a"), 11),
			maxBytes:    10,
			wantErr:     "file too large",
		},
		"db failure deletes object": {
			filename:    "report.txt",
			contentType: "text/plain",
			payload:     []byte("hello world"),
			maxBytes:    1024,
			repoErr:     errors.New("db down"),
			wantErr:     "create artifact record: db down",
			wantUpload:  true,
			wantDelete:  true,
		},
	}

	for _, tt := range tests {
		fileHeader := mustMakeFileHeader(t, tt.filename, tt.contentType, tt.payload)
		got, err := container.ServiceArtifactManager.Upload(context.Background(), user.ID, fileHeader)
		require.Error(t, err)
		require.Nil(t, got)
		require.Contains(t, err.Error(), tt.wantErr)
	}
}

func TestServiceUploadPositive(t *testing.T) {
	// given
	container := testhelpers.GetClean(t)
	user := seed.NewUserBuilder().PopulateTests(t, container)

	tests := map[string]testCase{
		"ok text file": {
			filename:       "report.txt",
			contentType:    "text/plain",
			payload:        []byte("hello world"),
			maxBytes:       1024,
			wantUpload:     true,
			wantSafeName:   "report.txt",
			wantDetectMIME: "text/plain; charset=utf-8",
		},
		"malicious filename sanitized": {
			filename:       "..\\..//evil\x00name?.pdf",
			contentType:    "application/pdf",
			payload:        []byte("%PDF-1.4\nbody"),
			maxBytes:       1024,
			wantUpload:     true,
			wantSafeName:   "....__evilname_.pdf",
			wantDetectMIME: "application/pdf",
		},
		"content type spoofing detected": {
			filename:       "image.png",
			contentType:    "image/png",
			payload:        []byte("%PDF-1.7\nfake png"),
			maxBytes:       1024,
			wantUpload:     true,
			wantSafeName:   "image.png",
			wantDetectMIME: "application/pdf",
		},
	}
	for _, tt := range tests {
		fileHeader := mustMakeFileHeader(t, tt.filename, tt.contentType, tt.payload)
		got, err := container.ServiceArtifactManager.Upload(context.Background(), user.ID, fileHeader)
		require.NoError(t, err)
		require.NotNil(t, got)

		require.Equal(t, user.ID, got.OwnerID)
		require.Equal(t, tt.wantSafeName, got.OriginalName)
		require.Equal(t, tt.contentType, got.DeclaredMIME)
		require.Equal(t, tt.wantDetectMIME, got.DetectedMIME)
		require.Equal(t, int64(len(tt.payload)), got.ByteSize)
		require.Equal(t, entities.ArtifactStorageS3, got.Storage)
		require.Equal(t, container.Cfg.S3.Bucket, got.Bucket)
		require.NotEmpty(t, got.ObjectKey)

		sum := sha256.Sum256(tt.payload)
		require.Equal(t, hex.EncodeToString(sum[:]), got.SHA256Hex)

		artifact, err := container.Repo.GetArtifactByID(container.Ctx, user.ID, got.ID) // should exist in DB
		require.NoError(t, err)
		require.NotNil(t, artifact)
	}
}

func mustMakeFileHeader(t *testing.T, filename, contentType string, payload []byte) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	partHeader.Set("Content-Type", contentType)

	part, err := w.CreatePart(partHeader)
	require.NoError(t, err)

	_, err = part.Write(payload)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	err = req.ParseMultipartForm(int64(len(payload)) + 1024)
	require.NoError(t, err)

	fhs := req.MultipartForm.File["file"]
	require.Len(t, fhs, 1)

	return fhs[0]
}
