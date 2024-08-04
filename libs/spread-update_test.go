package libs

import (
	"testing"
	"time"
)

// func TestUpdateCell(t *testing.T) error {
// 	t.Log("TestUpdateCell")
// 	return nil
// }

func TestUpload(t *testing.T) {
	pw, _, page, err := newPage(false)
	if err != nil {
		t.Fatal(err)
	}
	defer pwClose(pw, page)

	// test upload
	if _, err := page.Goto("https://www.filestack.com/"); err != nil {
		t.Fatal(err)
	}

	var files = []string{
		`4-twitter.mp4`,
	}

	inputFiles, err := filesToInputFiles(files)
	if err != nil {
		t.Fatal(err)
	}

	if err := page.Locator("input[id='fsp-fileUpload']").SetInputFiles(inputFiles); err != nil {
		t.Fatal(err)
	}

	if err := page.Locator("span[data-e2e='upload']").Tap(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Minute * 2)

	if err := Screenshot(page, `screenshot.png`); err != nil {
		t.Fatal(err)
	}
}
