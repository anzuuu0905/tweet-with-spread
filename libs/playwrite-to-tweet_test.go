package libs

import (
	"testing"
)

const (
	IS_TWITTER_POST = false
	POSTMSG         = ``
)

func TestLogin(t *testing.T) {
	accountID := ""
	password := ""

	files := []string{`1.png`, `2.png`}

	// f, err := os.Open(files[0])
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer f.Close()

	// info, err := f.Stat()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// fmt.Printf("%#v", info)
	with_files := true
	if err := TweetsToGUI(IS_TWITTER_POST, with_files, accountID, password, POSTMSG, files); err != nil {
		t.Fatal(err)
	}
}
