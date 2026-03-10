package utils_test

import (
	"lazarus/internal/utils"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeString(t *testing.T) {
	tests := map[string]string{
		"```json\n{\"a\":1}\n```":               "{\"a\":1}",
		"Here is the result:\n```json\n{}\n```": "{}",
		"\uFEFF```text\nhello\n```":             "hello",
		"   simple text   ":                     "simple text",
	}

	for in, want := range tests {
		require.Equal(t, want, utils.SanitizeResponse(in))
	}
}

func TestSanitizeResponseJSON(t *testing.T) {
	tests := map[string]string{
		`{"a":1}`:                           `{"a":1}`,
		`here is result: {"a":1}`:           `{"a":1}`,
		`{"a":1} done`:                      `{"a":1}`,
		`before {"a":1} after`:              `{"a":1}`,
		"```json\n{\"a\":1}\n```":           `{"a":1}`,
		`prefix {"a":{"b":2},"c":3} suffix`: `{"a":{"b":2},"c":3}`,
		`x {"a":1} y {"b":2} z`:             `{"a":1} y {"b":2}`,
		`abc}`:                              `abc}`,
		`{"a":1`:                            `{"a":1`,
		``:                                  ``,
		`hello world`:                       `hello world`,
		`[{"a":1}]`:                         `[{"a":1}]`,
		`abc { not json } tail {"a":1}`:     `{ not json } tail {"a":1}`,
	}

	for in, want := range tests {
		require.Equal(t, want, utils.SanitizeResponseJSON(in))
	}
}

func TestNormalizeLabName(t *testing.T) {
	tests := map[string]string{
		"":                       "",
		"Albumin":                "albumin",
		"ALB (Albumin)":          "alb",
		"Serum Albumin":          "albumin",
		"Plasma Glucose":         "glucose",
		"Blood Urea Nitrogen":    "urea nitrogen",
		"Total Protein":          "protein",
		"Serum Plasma Glucose":   "glucose",
		"Blood Total Hemoglobin": "hemoglobin",
		"  Serum   Albumin  ":    "albumin",
		"ALB   (Albumin)   Test": "alb test",

		"Serum (something)":                "",
		"Total (something)":                "",
		"Blood (Whole Blood) CBC":          "cbc",
		"serum plasma blood total calcium": "calcium",

		"Serum ":   "",
		"Serum   ": "",
		" Serum ":  "",
	}

	for in, want := range tests {
		require.Equal(t, want, utils.NormalizeLabName(in))
	}
	require.Equal(t, "", utils.NormalizeLabName(""))
}
