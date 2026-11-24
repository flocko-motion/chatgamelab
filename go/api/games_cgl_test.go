package api

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"cgl/obj"
	"cgl/picture"

	"gopkg.in/yaml.v3"
)

// unzip reads a zip's files by name, with how each was stored.
func unzip(t *testing.T, body []byte) (map[string][]byte, map[string]uint16) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	files, methods := map[string][]byte{}, map[string]uint16{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		files[f.Name], _ = io.ReadAll(rc)
		methods[f.Name] = f.Method
		rc.Close()
	}
	return files, methods
}

// A game's zip holds its game file, as the YAML export writes it, and its
// cover as it was stored.
func TestGameZipHoldsTheFileAndTheCover(t *testing.T) {
	file := obj.GameFile{Name: "Lighthouse", Description: "Keep the light", SystemMessageScenario: "A storm.\nA second line.", Genre: "adventure"}
	cover := []byte{0xFF, 0xD8, 0xFF, 1, 2, 3}
	body, err := gameZip(file, cover)
	if err != nil {
		t.Fatal(err)
	}
	files, methods := unzip(t, body)
	if len(files) != 3 {
		t.Fatalf("files %v, want game.yaml, cover.jpg and cover.sig", files)
	}
	if !picture.Signed(files[zipCover], string(files[zipCoverSig])) {
		t.Errorf("cover.sig %q does not sign the cover", files[zipCoverSig])
	}
	var back map[string]any
	if err := yaml.Unmarshal(files[zipGameFile], &back); err != nil {
		t.Fatalf("game.yaml does not read: %v", err)
	}
	if back["name"] != "Lighthouse" || back["system_message_scenario"] != "A storm.\nA second line." || back["genre"] != "adventure" {
		t.Errorf("game.yaml = %s", files[zipGameFile])
	}
	if bytes.Contains(files[zipGameFile], []byte(`"name"`)) {
		t.Errorf("game.yaml is JSON-styled, not block YAML: %s", files[zipGameFile])
	}
	if !bytes.Equal(files[zipCover], cover) || methods[zipCover] != zip.Store {
		t.Errorf("cover.jpg = %v (method %d)", files[zipCover], methods[zipCover])
	}
}

// A game without a cover has a zip without one.
func TestGameZipWithoutACover(t *testing.T) {
	body, err := gameZip(obj.GameFile{Name: "Bare"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if files, _ := unzip(t, body); len(files) != 1 || files[zipGameFile] == nil {
		t.Errorf("files %v, want game.yaml alone", files)
	}
}
