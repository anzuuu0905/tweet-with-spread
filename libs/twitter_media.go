package libs

import (
	"encoding/base64"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/rs/zerolog/log"
)

// TweetUpload メディアアップロード -> メディアIDを返す
// Twitter v1.1 API
func TweetUpload(account Box, files []string) []string {
	log.Debug().Str("function", "TweetUpload").Msgf("get files: %dfiles, %v", len(files), files)
	if len(files) == 0 {
		return nil
	}

	_, api := newClientV1(account)

	var (
		medias []string
	)
	for i := 0; i < len(files); i++ {
		// メディアファイルのみを抽出
		if strings.HasSuffix(files[i], ".jpg") ||
			strings.HasSuffix(files[i], ".png") ||
			strings.HasSuffix(files[i], ".gif") ||
			strings.HasSuffix(files[i], ".webp") {
			// メディアアップロード
			base64Str, err := ToBase64(files[i])
			if err != nil {
				log.Err(err).Msgf("failed to convert to base64, %s", files[i])
				continue
			}
			media, err := api.UploadMedia(base64Str)
			if err != nil {
				log.Err(err).Msgf("%v > failed to upload media, %s", err, files[i])
				continue
			}
			medias = append(medias, media.MediaIDString)
		} else if strings.HasSuffix(files[i], ".mp4") ||
			strings.HasSuffix(files[i], ".mov") {
			// Video upload
			media, err := UploadVideo(*api, files[i])
			if err != nil {
				log.Err(err).Msgf("failed to upload video, %s", files[i])
				continue
			}
			medias = append(medias, media)
		}
	}

	log.Debug().Msgf("file upload to twitter, media_ids: %v", medias)

	return medias
}

func ToBase64(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func UploadVideo(api anaconda.TwitterApi, filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		return "", err
	}

	fileType := "video/mp4"
	if strings.HasSuffix(filename, ".mov") {
		fileType = "video/quicktime"
	}

	media, err := api.UploadVideoInit(int(finfo.Size()), fileType)
	if err != nil {
		return "", err
	}

	log.Debug().Msgf("video upload to twitter, %v, %d", fileType, finfo.Size())

	// チャンクサイズ（例：5MB）
	chunkSize := 1024 * 1024
	buffer := make([]byte, chunkSize)
	var segmentIndex int

	for {
		bytesRead, err := f.Read(buffer)
		if err != nil && err != io.EOF {
			return "", SetError(err, "failed to read file")
		}
		if bytesRead == 0 {
			break
		}

		if err := api.UploadVideoAppend(media.MediaIDString, segmentIndex, base64.StdEncoding.EncodeToString(buffer[:bytesRead])); err != nil {
			return "", SetError(err, "failed to upload video append")
		}
		segmentIndex++
	}

	// アップロードの完了
	result, err := api.UploadVideoFinalize(media.MediaIDString)
	if err != nil {
		return "", SetError(err, "failed to upload video finalize")
	}

	time.Sleep(10 * time.Second)
	log.Debug().Msgf("video uploaded to twitter, %s", media.MediaIDString)
	log.Debug().Msgf("%v, %d", result.Video.VideoType, result.Size)

	return media.MediaIDString, nil
}
