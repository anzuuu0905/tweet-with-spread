/*
Subsets models.go は、ユーザーが利用するサブセットのモデルを提供します。

このパッケージは、ユーザーが利用するサブセットのモデルを提供します。
cmd/User596E9F4におけるオンデマンドな処理を行うためのモデルを提供します。
処理群
- Twitterアカウント
- Twitterアカウントが包括する投稿群

*/

package subsets

// TwitterAccount は、Twitterアカウントを表します。
type TwitterAccount struct {
	Index          int    `csv:"index"`
	TwitterID      string `csv:"twitter_id"`
	TwitterName    string `csv:"twitter_name"`
	Password       string `csv:"password"`
	SpreadID       string `csv:"spread_id"`
	ConsumerKey    string `csv:"consumer_key"`
	ConsumerSecret string `csv:"consumer_secret"`
	AccessToken    string `csv:"access_token"`
	SecretToken    string `csv:"secret_token"`
	BearerToken    string `csv:"bearer_token"`
	Subscribed     int    `csv:"subscribed"`


	// 時間指定での投稿を行う場合の項目
	Hours   string `csv:"hours"`
	Minutes string `csv:"minutes"`

	// 以下は現行未使用
	TermDays int `csv:"term_days"`
	// Tel   string `csv:"tel"`
}

// Keys は、TwitterAccountのキーを返します。
// 必要な処理を行うためのインターフェースを実装します。
func (a TwitterAccount) Keys() (id, consumerKey, consumerSecret, accessToken, accessTokenSecret string) {
	return a.TwitterID, a.ConsumerKey, a.ConsumerSecret, a.AccessToken, a.SecretToken
}

// TwitterTweet は、Twitterアカウントが包括する投稿群を表します。
type TwitterTweet struct {
	Index     int    `csv:"index"`
	TwitterID string `csv:"twitter_id"`
	Text      string `csv:"text"`
	// Files
	Files     string `csv:"-"`
	File1     string `csv:"file1"`
	File2     string `csv:"file2"`
	File3     string `csv:"file3"`
	File4     string `csv:"file4"`
	WithFiles int    `csv:"with_files"`
	

	// 分岐処理用項目
	Kind     int `csv:"kind"`
	Type     int `csv:"type"`
	Checked  int `csv:"checked"`
	Priority int `csv:"priority"`

	// 上書き部分
	Count    int    `csv:"count"`
	TweetURL string `csv:"tweet_url"`
	LastDate string `csv:"last_date"` // 形式: YYYY/MM/DD HH:MM:SS
}
