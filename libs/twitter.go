/*
only manage/create, lookup/me with Twitter API Free Plan
*/

package libs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/michimani/gotwi"
	"github.com/michimani/gotwi/tweet/managetweet"
	mtypes "github.com/michimani/gotwi/tweet/managetweet/types"
	"github.com/rs/zerolog/log"
)

type Box interface {
	Keys() (id, consumerKey, consumerSecret, accessToken, accessTokenSecret string)
}

type TwitterAccount struct {
	ID                string `csv:"id"`
	ConsumerKey       string `csv:"consumer_key"`
	ConsumerSecret    string `csv:"consumer_secret"`
	AccessToken       string `csv:"access_token"`
	AccessTokenSecret string `csv:"access_token_secret"`
}

func (a TwitterAccount) Keys() (id, consumerKey, consumerSecret, accessToken, accessTokenSecret string) {
	return a.ID, a.ConsumerKey, a.ConsumerSecret, a.AccessToken, a.AccessTokenSecret
}

// LoggingInterceptor リクエスト前後に処理を追加する
// ‐ ヘッダー情報を取得する
type LoggingInterceptor struct {
	Transport http.RoundTripper
	XLimit    int
	XResetSec int
}

func NewLoggingInterceptor() *LoggingInterceptor {
	return &LoggingInterceptor{
		Transport: http.DefaultTransport,
		XLimit:    10,
		XResetSec: 15 * 60,
	}
}

func (li *LoggingInterceptor) RoundTrip(req *http.Request) (*http.Response, error) {
	// リクエスト前のロジック
	// API レートリミットでリクエストを制限する
	// API Limitが少ないときにエラーを出し続けると、API Limitが復活するための情報を得られないため、
	if li.XLimit != 0 && li.XLimit <= 1 {
		// 待機し過ぎにならないように、リクエストを制限する
		time.Sleep(time.Duration(15) * time.Second)
		li.XResetSec -= 15
		if li.XResetSec <= 0 {
			// リセット時間が過ぎた場合は、リミットをリセットする
			// API Informationを取得するために最低限のリクエストを許可する
			li.XLimit = 10
		}
		return nil, fmt.Errorf("rate limit exceeded, api limit: %d, reset in: %d", li.XLimit, li.XResetSec)
	}

	// リクエスト実行
	resp, err := li.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// HeaderからAPI レートリミット情報を取得する
	s := resp.Header.Get("x-rate-limit-remaining")
	li.XLimit, _ = strconv.Atoi(s)
	s = resp.Header.Get("x-rate-limit-reset")
	li.XResetSec, _ = strconv.Atoi(s)
	log.Info().Msgf("x-rate-limit-remaining: %d, x-rate-limit-reset: %d", li.XLimit, li.XResetSec)

	return resp, nil
}

func (li *LoggingInterceptor) Tweeting(is_post bool, account Box, req *mtypes.CreateInput) (*mtypes.CreateOutput, error) {
	id, consumersurKey, consumersurSecret, accessToken, accessTokenSecret := account.Keys()
	if err := os.Setenv("GOTWI_API_KEY", consumersurKey); err != nil {
		return nil, SetError(err, errors.New("failed to set env GOTWI_API_KEY"))
	}
	if err := os.Setenv("GOTWI_API_KEY_SECRET", consumersurSecret); err != nil {
		return nil, SetError(err, errors.New("failed to set env GOTWI_API_KEY_SECRET"))
	}

	in := &gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           accessToken,
		OAuthTokenSecret:     accessTokenSecret,
	}

	fmt.Printf("---------\napi key: %s, secret: %s, access token: %s, access secret: %s\n----------\n", os.Getenv("GOTWI_API_KEY"), os.Getenv("GOTWI_API_KEY_SECRET"), accessToken, accessTokenSecret)

	c, err := gotwi.NewClient(in)
	if err != nil {
		return nil, SetError(err, errors.New("failed to create a new client"))
	}

	// トランスポートを設定しリクエスト前後に処理を追加する
	// ‐ ヘッダー情報を取得する
	c.Client.Transport = li

	if !is_post {
		return nil, fmt.Errorf("[定数設定] not post, program constants limit posting privileges, request, %s -> %s", id, *req.Text)
	}
	ctx := context.Background()
	res, err := managetweet.Create(ctx, c, req)
	if err != nil {
		return nil, SetError(err, errors.New("failed to tweet"))
	}

	log.Debug().Msgf("success tweet: %s, %s", *res.Data.ID, *res.Data.Text)

	return res, nil
}

func Delete(account Box, req *mtypes.DeleteInput) error {
	id, consumersurKey, consumersurSecret, accessToken, accessTokenSecret := account.Keys()
	if err := os.Setenv("GOTWI_API_KEY", consumersurKey); err != nil {
		return SetError(err, errors.New("failed to set env GOTWI_API_KEY"))
	}
	if err := os.Setenv("GOTWI_API_KEY_SECRET", consumersurSecret); err != nil {
		return SetError(err, errors.New("failed to set env GOTWI_API_KEY_SECRET"))
	}

	in := &gotwi.NewClientInput{
		AuthenticationMethod: gotwi.AuthenMethodOAuth1UserContext,
		OAuthToken:           accessToken,
		OAuthTokenSecret:     accessTokenSecret,
	}

	c, err := gotwi.NewClient(in)
	if err != nil {
		return SetError(err, errors.New("failed to create a new client"))
	}

	ctx := context.Background()
	res, err := managetweet.Delete(ctx, c, req)
	if err != nil {
		return SetError(err, errors.New("failed to tweet"))
	}

	log.Debug().Msgf("account: %s -> deleted success: %t", id, *res.Data.Deleted)

	return nil
}
