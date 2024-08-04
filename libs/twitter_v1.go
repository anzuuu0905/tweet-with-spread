package libs

import (
	"net/url"

	"github.com/ChimeraCoder/anaconda"
)

func newClientV1(account Box) (string, *anaconda.TwitterApi) {
	twitterID, consumerKey, consumerSecret, accessToken, accessTokenSecret := account.Keys()

	anaconda.SetConsumerKey(consumerKey)
	anaconda.SetConsumerSecret(consumerSecret)
	api := anaconda.NewTwitterApi(accessToken, accessTokenSecret)

	return twitterID, api
}

// GetTweets Twitterのユーザータイムラインを取得
func GetTweets(account Box) ([]anaconda.Tweet, error) {
	twitterID, api := newClientV1(account)

	u := url.Values{}
	u.Add("screen_name", twitterID)
	tweets, err := api.GetUserTimeline(u)
	if err != nil {
		return nil, SetError(err, "get user timeline error")
	}

	return tweets, nil
}
