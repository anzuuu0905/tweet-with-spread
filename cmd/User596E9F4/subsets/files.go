package subsets

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"tweet-with-spread/libs"

	"github.com/michimani/gotwi/tweet/managetweet/types"
	"github.com/rs/zerolog/log"
)

// RequestCreateTweet API Twitter投稿リクエストを作成する
func RequestCreateTweet(account TwitterAccount, tweet *TwitterTweet, files []string) (*types.CreateInput, error) {
	req := &types.CreateInput{
		Text: &tweet.Text,
	}
	mediaIDs := libs.TweetUpload(account, files)
	if len(mediaIDs) != 0 && mediaIDs != nil {
		req.Media = &types.CreateInputMedia{
			MediaIDs: mediaIDs,
		}
	}

	// メディアアップロードに失敗した場合は、画像なしで投稿するか判断
	if len(files) != len(mediaIDs) {
		log.Error().Err(fmt.Errorf("files: %v -> media_ids: %v", files, mediaIDs)).Str("function", "RequestCreateTweet").Msgf("failed to upload media, %s", account.TwitterID)
		if tweet.WithFiles == 1 {
			return nil, fmt.Errorf("failed to upload media, %s", account.TwitterID)
		}
	}
	log.Debug().Str("function", "RequestCreateTweet").Msgf("tweet request: %+v", req)

	return req, nil
}

// Tofiles Spread項目からfiles []stringを生成する
func (p *TwitterTweet) Tofiles(cred []byte) []string {
	var files []string
	for _, file := range []string{p.File1, p.File2, p.File3, p.File4} {
		// 空文字列は無視
		if file != "" {
			// DriveURLからファイルをダウンロードし、一時保存先を返す。DriveURLでない場合はそのまま返す
			file = DriveToFile(cred, file)
			files = append(files, file)
		}
	}
	return files
}

// // DriveToFile GetDriveFile GoogleDriveAPIを使用してDriveURLからファイルをダウンロードし、一時保存先を返す。DriveURLでない場合はそのまま返す
func DriveToFile(cred []byte, file string) string {
	// DriveURLからファイルIDを取得
	// 取得できなければそのまま返す
	fileID, err := libs.GetFileIDFromDriveURL(file)
	if err != nil {
		log.Debug().Msgf("failed to get fileID: %s", err)
		return file
	}

	// FileIDが取得できれば
	// Driveファイルをダウンロード
	b, err := libs.GetDriveFile(cred, fileID)
	if err != nil {
		log.Err(err).Msgf("failed to get drive file")
		return file
	}

	// ファイルの種類を判別する
	fileTypes := map[string]string{
		"FFD8FF":   "jpg",
		"FFD8DDE0": "jpeg",
		"89504E47": "png",
		"47494638": "gif",
		"000000":   "mov",
		"66747970": "mp4",
		"464C56":   "flv",
		"52494646": "webp",
		// 適宜ファイルタイプに対するマッピングも追加
	}
	fileExtension := ""
	for magic, ext := range fileTypes {
		if strings.HasPrefix(fmt.Sprintf("%X", b), magic) {
			fileExtension = ext
			break
		}
	}

	if fileExtension == "" {
		log.Error().Msg("unknown file type, cannot save file")
		return ""
	}

	filename := fmt.Sprintf("%s.%s", fileID, fileExtension)
	// ファイルを保存
	saveTo := filepath.Join(TEMPORARYDIR, filename)
	if err := libs.SaveFile(b, saveTo); err != nil {
		log.Err(err).Msgf("failed to save file")
		return file
	}

	// CrossPlatformでのファイルパスを返す
	relativePath := filepath.Join(strings.Split(saveTo, "/")...)
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		return saveTo
	}

	log.Debug().Msgf("file saved abs path: %s", absPath)

	return absPath
}

// StrToIntSlice 文字列を数値に変換する
//   - 例: "1,2,3" -> []int{1, 2, 3}
func StrToIntSlice(str string) []int {
	// 文字列からSliceに
	slice := strings.Split(str, ",")

	var result []int
	for _, v := range slice {
		i, err := strconv.Atoi(v)
		if err != nil {
			continue
		}

		result = append(result, i)
	}
	return result
}
