package libs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// ID2TwitterURL() string
func ID2TwitterURL(id string) string {
	return fmt.Sprintf("https://twitter.com/i/web/status/%s", id)
}

// SaveFile ファイルを保存する
func SaveFile(data []byte, saveTo string) error {
	// ファイルを保存する
	// ファイルが存在する場合は上書きする
	// ファイルが存在しない場合は新規作成する
	if err := os.WriteFile(saveTo, data, 0644); err != nil {
		return SetError(err, "failed to save file")
	}

	return nil
}

// CleanDir ディレクトリ内のファイルをクリーンアップする
func CleanDir(dir string) error {
	// ディレクトリをクリーンアップする
	files, err := os.ReadDir(dir)
	if err != nil {
		return SetError(err, "failed to read directory")
	}

	var errs []error
	for i := 0; i < len(files); i++ {
		fileto := filepath.Join(dir, files[i].Name())
		errs = append(errs, os.RemoveAll(fileto))
	}

	// ファイル削除でエラーが発生した場合は一時的に保留し処理を続行、すべてのファイルを削除する
	// その後エラーを処理する
	tempErr := errors.Join(errs...)
	if tempErr != nil {
		return SetError(err, "failed to remove file")
	}

	return nil
}

// GetDriveFile GoogleDriveAPIを使用してファイルをダウンロードする
func GetDriveFile(cred []byte, fileID string) ([]byte, error) {
	// GoogelDriveAPI ファイルを取得
	config, err := google.JWTConfigFromJSON(cred, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// google clientを作成
	ctx := context.Background()
	client := config.Client(ctx)
	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, SetError(err, "failed to create google drive service")
	}

	file, err := srv.Files.Get(fileID).Download()
	if err != nil {
		return nil, SetError(err, "failed to get file")
	}
	defer file.Body.Close()

	data, err := io.ReadAll(file.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetFileIDFromDriveURL url `https://drive.google.com/file/d/<FileID>/view`からFileIDを取得する
func GetFileIDFromDriveURL(url string) (driveFileID string, err error) {
	if !strings.Contains(url, "https://drive.google.com/file/d/") {
		return "", errors.New("invalid google drive file url")
	}

	// 二段階に分ける
	// why: https://がない場合を考慮
	s := strings.Replace(url, "https://", "", 1)
	s = strings.Replace(s, "drive.google.com/file/d/", "", 1)
	str := strings.Split(s, "/")

	// str lengthが1である場合はそのまま返し、2の場合は1爪を返す。それ以外はエラー
	if len(str) == 1 || len(str) == 2 {
		return str[0], nil
	}

	return "", errors.New("invalid google drive file url")
}

// OtherDriveFileUrlToFile
// 公開されたGoogleDriveFileURLからUploadFileを生成する
// https://drive.google.com/file/d/<FileID>/view?usp=sharing
// これをダウンロードリンクに変換するには、以下の形式に変更します：
// https://drive.google.com/uc?export=download&id=<FileID>
func OtherDriveFileUrlToFile(getURL, saveDir string) (string, error) {
	if !strings.HasSuffix(getURL, "/view?usp=sharing") {
		return "", errors.New("invalid google drive file url")
	}

	// FileIDを取得
	fileID := strings.Replace(getURL, "/view?usp=sharing", "", 1)
	fileID = strings.Replace(fileID, "https://drive.google.com/file/d/", "", 1)

	// HTTP GET リクエストを発行してファイルを取得
	resp, err := http.Get(getURL)
	if err != nil {
		return "", SetError(err, "failed to get file")
	}
	defer resp.Body.Close()

	// 空のファイルを作成
	filepath := saveDir + "/" + fileID
	out, err := os.Create(saveDir)
	if err != nil {
		return "", SetError(err, "failed to create file")
	}
	defer out.Close()

	// レスポンスの内容をファイルに書き込む
	_, err = io.Copy(out, resp.Body)
	return filepath, SetError(err, "failed to write file")
}
