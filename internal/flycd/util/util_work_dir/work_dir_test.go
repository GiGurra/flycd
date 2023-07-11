package util_work_dir

import "testing"

func TestCopyFiles(t *testing.T) {
	tempDir1, err := NewTempDir("test", "")
	if err != nil {
		t.Fatalf("error creating temp dir: %s", err)
	}
	defer tempDir1.RemoveAll()

	tempDir2, err := NewTempDir("test", "")
	if err != nil {
		t.Fatalf("error creating temp dir: %s", err)
	}
	defer tempDir2.RemoveAll()

	// Create a file in tempDir1
	err = tempDir1.WriteFile("test.txt", "test")
	if err != nil {
		t.Fatalf("error writing file: %s", err)
	}

	// Copy the file to tempDir2
	err = tempDir1.CopyContentsTo(tempDir2)
	if err != nil {
		t.Fatalf("error copying files: %s", err)
	}

	// Check that the file exists in tempDir2
	contents, err := tempDir2.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("error reading file: %s", err)
	}

	if contents != "test" {
		t.Fatalf("file contents incorrect: %s", contents)
	}
}
