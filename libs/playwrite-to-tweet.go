package libs

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/rs/zerolog/log"
)

const (
	// /i/flow/login
	TWITTER    = "https://twitter.com"
	TWITTERPRO = "https://pro.twitter.com"

	// ファイルアップロード待ち時間
	// Important! : インスタンスやネットワークの状況によって変更してください
	MAXWAITFORUPLOAD = 120

	is_debug = false
)

// TweetsToGUI Login & Tweet
// Two-step verification is not supported.
// - newPage()
// - login()
// - post()
func TweetsToGUI(is_post, with_files bool, accountID, password, postMessage string, fileAbsolutePaths interface{}) error {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	// create new page with context
	pw, browser, page, err := newPage(is_post)
	if err != nil {
		return SetError(err, "could not create new page")
	}
	defer pwClose(pw, page)

	if is_debug { // デバッグ用: javascriptの挙動を知るためのログを出力
		page.On("console", func(console playwright.ConsoleMessage) {
			log.Debug().Msgf("console message: %s", console.Text())
		})
		page.On("dialog", func(dialog playwright.Dialog) {
			log.Debug().Msgf("dialog message: %s", dialog.Message())
		})

		page.On("response", func(response playwright.Response) {
			// レスポンスのURL
			url := response.URL()
			// ステータスコード
			status := response.Status()
			// ステータスメッセージ
			statusText := response.StatusText()
			// // ヘッダー（注意: ヘッダーはmap形式）
			// headers, err := response.AllHeaders() // エラーハンドリングは適宜行ってください
			// if err != nil {
			// 	log.Error().Err(err)
			// 	return
			// }

			// これらの情報をログに出力
			log.Printf("Response URL: %s", url)
			log.Printf("Status: %d (%s)", status, statusText)

			if strings.HasSuffix(url, ".json") {
				log.Printf("start request, %s", url)
				c := http.Client{
					Timeout: time.Second * 2,
				}

				log.Debug().Msgf("request create! %s", url)
				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					log.Error().Err(err).Msg("could not create request")
					return
				}

				log.Debug().Msgf("request do!")

				res, err := c.Do(req)
				if err != nil {
					log.Error().Err(err).Msg("could not get response")
					res.Body.Close()
					return
				}
				if res.StatusCode != 200 {
					log.Error().Msgf("could not get response, status code: %d", res.StatusCode)
					res.Body.Close()
					return
				}

				log.Debug().Msgf("response comming!")

				b, err := io.ReadAll(res.Body)
				if err != nil {
					log.Error().Err(err).Msg("could not read response body")
					return
				}
				res.Body.Close()

				log.Debug().Msgf("response body: %s", string(b))

				spliturl := strings.Split(url, "/")
				f, err := os.Create(fmt.Sprintf("./jsons/%s", spliturl[len(spliturl)-1]))
				if err != nil {
					log.Error().Err(err).Msg("could not create file")
					return
				}
				if _, err := f.Write(b); err != nil {
					log.Error().Err(err).Msg("could not write file")
					return
				}

				if err := f.Close(); err != nil {
					log.Error().Err(err).Msg("could not close file")
					return
				}
			}

		})
	}

	u, err := url.Parse(TWITTER)
	if err != nil {
		return SetError(err, "could not parse url")
	}
	u.Path = "/i/flow/login"
	log.Debug().Msgf("target url: %s", u.String())

	if _, err = page.Goto(u.String()); err != nil {
		if _, err := page.Reload(); err != nil {
			return SetError(err, fmt.Errorf("%v, could not goto %s", err, u.String()))
		}
	}

	// ログインセクション
	// input UserID/TwitterID/TEL/Email
	if err := login(page, accountID, password, r); err != nil {
		return SetError(err, "could not twitter login")
	}

	if is_debug {
		contexts := browser.Contexts()
		for _, c := range contexts {
			cookies, _ := c.Cookies(u.String())
			for _, cookie := range cookies {
				log.Debug().Msgf("cookie: %v", cookie)
			}
		}
	}

	// to pro.twitter.com
	// u, _ = url.Parse(TWITTERPRO)
	// why Proページ変異なく同様の動作をするため、コメントアウト

	u.Path = "/home"
	if _, err = page.Goto(u.String(), playwright.PageGotoOptions{
		Timeout: playwright.Float(60000),
	}); err != nil {
		return SetError(err, "could not goto "+u.String())
	}

	time.Sleep(time.Millisecond * time.Duration(millisec(r)))

	// 投稿セクション
	if !is_post {
		return fmt.Errorf("[定数設定] not post for gui, program constants limit posting privileges, request, %v", postMessage)
	}
	if err := post(with_files, page, postMessage, fileAbsolutePaths.([]string), r); err != nil {
		return SetError(err, "could not post")
	}

	log.Info().Msgf("tweeted: %s", accountID)

	return nil
}

// login ログインセクション: GUIや仕様が変わった場合はこの関数を変更してください
func login(page playwright.Page, accountID, password string, r *rand.Rand) error {
	// if err := Screenshot(page, "login-id.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }

	if err := page.Locator("input[type='text']").Fill(accountID); err != nil {
		return SetError(err, "could not fill to account input")
	}

	time.Sleep(time.Millisecond * time.Duration(millisec(r)))

	if err := page.Locator(`xpath=//span[text()='次へ']`).Tap(); err != nil {
		return SetError(err, "could not click to 次へ button")
	}

	// if err := Screenshot(page, "login-password.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }

	// input Password
	if err := page.Locator("input[type='password']").Fill(password); err != nil {
		return SetError(err, "could not fill to password input")
	}

	time.Sleep(time.Millisecond * time.Duration(millisec(r)))

	if err := page.Locator("[data-testid='LoginForm_Login_Button']").Nth(0).Tap(); err != nil {
		return SetError(err, "could not click to login button")
	}

	time.Sleep(time.Millisecond * time.Duration(millisec(r)))

	return nil
}

// post 投稿セクション: GUIや仕様が変わった場合はこの関数を変更してください
func post(with_files bool, page playwright.Page, msg string, files []string, r *rand.Rand) error {
	// if err := Screenshot(page, "post-start.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }

	time.Sleep(time.Millisecond * time.Duration(millisec(r)))
	// contenteditable属性を持つ要素にテキストを入力
	isVisible, err := page.Locator("[data-testid='tweetTextarea_0']").IsVisible()
	if err != nil {
		return SetError(err, "could not check the element is visible")
	}

	time.Sleep(10 * time.Second)

	if !isVisible {
		// 入力画面がなければ入力画面表示ボタンをタップ
		// 通知がある場合などに通知画面優先表示されるため対策
		if err := page.Locator("div[aria-label='ポストを作成']").Tap(); err != nil {
			log.Debug().Msgf("%v", SetError(err, "ok or could not tap to ポストを作成 element"))
		}
	}

	if err := page.Locator("[data-testid='tweetTextarea_0']").Fill(msg); err != nil {
		return SetError(err, "could not fill to tweet input")
	}

	// if err := Screenshot(page, "post-msg-fill.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }

	// ファイルをアップロード
	if err := uploadFiles(r, page, with_files, files); err != nil {
		return SetError(err, "could not upload files")
	}

	// if err := Screenshot(page, "post-upload.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }

	// ツイートボタンをクリック
	if err := page.Locator(`xpath=//span[text()='ポストする']`).Tap(); err != nil {
		return SetError(err, "could not click to post button")
	}

	time.Sleep(time.Millisecond * time.Duration(millisec(r)))

	return nil
}

// uploadFiles ファイルをアップロードする
func uploadFiles(r *rand.Rand, page playwright.Page, with_files bool, files []string) error {
	if len(files) == 0 {
		log.Debug().Msgf("no files to upload, files: %v", files)
		return nil
	}

	time.Sleep(time.Millisecond * time.Duration(millisec(r)))
	// if err := Screenshot(page, "screenshot-before.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }

	// GUIが求める型式に変更する
	inputFiles, err := filesToInputFiles(files)
	if err != nil {
		return SetError(err, "could not convert files to input files")
	}

	// ファイルをアップロード
	if err := page.Locator("input[data-testid='fileInput']").SetInputFiles(inputFiles, playwright.LocatorSetInputFilesOptions{
		NoWaitAfter: playwright.Bool(false),
		Timeout:     playwright.Float(60000),
	}); err != nil {
		if with_files { // ファイルの選択ができない場合、エラーを返す
			return SetError(err, "could not upload file")
		} else { // ファイルの投稿がなくても続行する
			log.Debug().Msgf("ok or could not upload file: %v", err)
		}
	}
	if err := page.Locator("input[accept='image/jpeg,image/png,image/webp,image/gif,video/mp4,video/quicktime']").SetInputFiles(inputFiles, playwright.LocatorSetInputFilesOptions{
		NoWaitAfter: playwright.Bool(false),
		Timeout:     playwright.Float(60000),
	}); err != nil {
		if with_files { // ファイルの選択ができない場合、エラーを返す
			return SetError(err, "could not upload file for type: file")
		} else { // ファイルの投稿がなくても続行する
			log.Debug().Msgf("ok or could not upload file type: %v", err)
		}
	}

	// for debug
	// pageContent, _ := page.Content()
	// // f, _ := os.Create("pageContent.html")
	// // f.Write([]byte(pageContent))
	// // f.Close()
	// if strings.Contains(pageContent, "一部の画像/動画を読み込めません。") {
	// 	if with_files { // ファイルの選択ができない場合、エラーを返す
	// 		return SetError(err, "could not upload file, 一部の画像/動画を読み込めません。")
	// 	} else { // ファイルの投稿がなくても続行する
	// 		log.Debug().Str("function", "post").Msgf("ok or could not upload file: %v", err)
	// 	}
	// }

	// if err := Screenshot(page, "screenshot.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }
	// ファイルのアップロード待ち
	// time.Sleep(1 * time.Minute)
	var (
		maxWaitSec = MAXWAITFORUPLOAD
		isOK       bool
	)
	for i := 0; i < maxWaitSec; i++ {
		isThere, err := page.Locator("div[data-testid='attachments']").IsVisible()
		if err != nil {
			return SetError(err, "could not check the element is visible")
		}

		// 投稿画像及び動画が表示された
		if isThere {
			isOK = true
			log.Debug().Int("gui upload wait sec", MAXWAITFORUPLOAD-i).Msg("ok or could not upload file")
			break
		}

		time.Sleep(time.Second)
	}
	if with_files { // ファイル必須ならば、ファイルの表示を確認してから判断する
		if !isOK {
			return SetError(fmt.Errorf("could not upload file, timeout: %d", maxWaitSec), "could not upload file")
		}
	}

	// if err := Screenshot(page, "success.png"); err != nil {
	// 	return SetError(err, "could not screenshot")
	// }

	return nil

}

// screenshot デバッグ用 ブラウザ動作でのスクリーンショットを撮る
func Screenshot(page playwright.Page, filename string) error {
	b, err := page.Screenshot()
	if err != nil {
		return SetError(err, "could not screenshot")
	}
	// ファイルへ保存
	f, err := os.Create(filename)
	if err != nil {
		return SetError(err, "could not create file")
	}
	defer f.Close()
	f.Write(b)

	return nil
}

// filesToInputFiles ファイルをアップロードするための目的の型式に変換する
func filesToInputFiles(files []string) ([]playwright.InputFile, error) {
	var inputFiles []playwright.InputFile
	for _, file := range files {
		name, buffer, err := readFile(file)
		if err != nil {
			log.Error().Err(err).Str("function", "filesToInputFiles")
			return nil, err
		}

		fileType := http.DetectContentType(buffer)
		if fileType == "application/octet-stream" {
			fileType = "video/quicktime"
		}

		log.Debug().Str("function", "filesToInputFiles").Msgf("file: %s, type: %s, byte size: %d", name, fileType, len(buffer))
		inputFiles = append(inputFiles, playwright.InputFile{
			Name:     name,
			MimeType: fileType,
			Buffer:   buffer,
		})
	}

	return inputFiles, nil
}

// readFile 小さいインスタンスでも大きいファイルを扱うため、チャンクで読み込む
func readFile(file string) (string, []byte, error) {
	f, err := os.Open(file)
	if err != nil {
		return "", nil, SetError(err, "could not open file")
	}
	defer f.Close()

	// チャンクで読み込みを行う
	var buffer []byte
	buf := make([]byte, 1024*1024) // 1MBのバッファ
	for {
		// ファイルからデータを読み込む
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return "", nil, err // 読み込み中にエラーが発生した場合
		}
		if n == 0 {
			break // ファイルの終端に達した場合、読み込みを終了
		}

		// 読み込んだデータをバッファに追加
		buffer = append(buffer, buf[:n]...)
	}

	return f.Name(), buffer, nil
}
