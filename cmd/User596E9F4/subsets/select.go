/*
	Selectors
	集合から指定の群を選別する
	- SelectTwitterAccounts: 投稿するアカウントを選別する
	- SelectTweet: 投稿するTweetsを選別する


	上記関数群を補助する子関数を実装しています。




*/

package subsets

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	// 日付のフォーマット
	// "2006/01/02 15:04:05" -> "YYYY/MM/DD HH:MM:SS"
	LAYOUT string = "2006/01/02 15:04:05"

	// ファイルの一時保存先
	// プログラムによりファイルは24時に削除される
	// why: ストレージを圧迫しないため
	TEMPORARYDIR = "./temp"
)

// SelectTwitterAccounts Twitter account listから投稿するべきアカウントを取得する
func SelectTwitterAccounts(t time.Time, sourceAccounts []TwitterAccount) ([]TwitterAccount, error) {
	if len(sourceAccounts) == 0 {
		return nil, errors.New("no accounts in list at the start")
	}

	// 現在時刻に合致するアカウントを選別
	log.Debug().Str("function", "SelectTwitterAccounts").Msgf("current time: %s", t.String())

	// 投稿するアカウントを選別
	var targetAccounts []TwitterAccount
	for i := 0; i < len(sourceAccounts); i++ {
		// サブスクライブを選別
		if sourceAccounts[i].Subscribed != 1 {
			log.Debug().Msgf("unsubscribed account: %s", sourceAccounts[i].TwitterID)
			continue
		}

		hours := StrToIntSlice(sourceAccounts[i].Hours)
		// 時間指定での投稿を行う場合の選別
		for j := 0; j < len(hours); j++ {
			// 指定時間総当たりで選別
			// 一致しなければ次の指定時間へ
			if t.Hour() != hours[j] {
				continue
			}

			// 時間合致確認後
			// 分指定での投稿を行う場合の選別
			minutes := StrToIntSlice(sourceAccounts[i].Minutes)
			for k := 0; k < len(minutes); k++ {
				if t.Minute() != minutes[k] {
					continue
				}
				targetAccounts = append(targetAccounts, sourceAccounts[i])
				// 一つでも合致したら次のアカウントへ
				break
			}
		}
	}

	if len(targetAccounts) == 0 {
		return nil, errors.New("no accounts in list")
	}

	return targetAccounts, nil
}

// SelectTweet 指定条件でTweetsを選択する
// ‐ 現行: 最後の投稿から指定日数経過したTweetsを抜粋
func SelectTweet(account TwitterAccount, tweets []TwitterTweet) (*TwitterTweet, error) {
	// 指定アカウントのTweetsを抜粋
	targetTweet, err := TweetsByAccount(account, tweets)
	if err != nil {
		return nil, err
	}

	// // 最後の投稿から指定日数経過したTweetsを抜粋
	targetTweet, err = TweetsByDate(account, targetTweet)
	if err != nil {
		return nil, err
	}

	// チェック有無でTweetsを抜粋
	targetTweet, err = TweetsByChecked(account, targetTweet)
	if err != nil {
		return nil, err
	}

	// PriorityでTweetsをソート
	targetTweet, err = SortByPriority(account, targetTweet)
	if err != nil {
		return nil, err
	}

	// CountでTweetsをソート
	targetTweet, err = TweetsByCount(account, targetTweet)
	if err != nil {
		return nil, err
	}

	// Tweetを一つ選択
	selectedTweet, err := SelectTweetsToOne(targetTweet)
	if err != nil {
		return nil, err
	}

	// 最終チェック
	if err := finalCheck(account, *selectedTweet); err != nil {
		return nil, err
	}

	return selectedTweet, nil
}

/*
	Select elements 投稿選択の条件分岐実装各項目
	- tweetsByAccount: 指定アカウントのTweetsを抜粋
	- tweetsByDate: 最後の投稿から指定日数経過したTweetsを抜粋
	- tweetsByChecked: チェック有無でTweetsを抜粋
	- sortByPriority: PriorityでTweetsをソート
	- tweetsByCount: CountでTweetsをソート

	Select One 複数の候補が残っていればランダム
	- SelectTweetsToOne: Tweetを一つ選択

	最終チェック
	- finalCheck: accountとtweetのTwitterIDが一致するか

	Checker
	- isExist 一つ以上のTweetsが存在するか
*/

func TweetsByAccount(account TwitterAccount, tweets []TwitterTweet) ([]TwitterTweet, error) {
	isThere, _, err := Exist(tweets)
	if err != nil || !isThere {
		return nil, err
	}
	// 一つであってもチェックする

	var targetTweet []TwitterTweet
	for i := 0; i < len(tweets); i++ {
		if tweets[i].TwitterID != account.TwitterID {
			continue
		}
		targetTweet = append(targetTweet, tweets[i])
	}
	if len(targetTweet) == 0 {
		return nil, errors.New("no tweet at the same id")
	}

	return targetTweet, nil
}

func TweetsByDate(account TwitterAccount, tweets []TwitterTweet) ([]TwitterTweet, error) {
	isThere, _, err := Exist(tweets)
	if err != nil || !isThere {
		return nil, err
	}
	// 一つであってもチェックする

	var selectedTweets []TwitterTweet
	borderDate := time.Now().AddDate(0, 0, -int(account.TermDays))
	for i := 0; i < len(tweets); i++ {
		// JTStimeから文字列で保存されたLastDateをtime型に変換、JSTとの時差を緩衝する
		// 失敗した場合は、ログを出力し次の処理を続行する
		t, err := AdjustDate(tweets[i].LastDate)
		if err != nil {
			log.Warn().Msgf("failed to adjust date [%s]: %s, ", tweets[i].LastDate, err)
			continue
		}

		// tはborderDateより過去である
		if t.Before(borderDate) {
			selectedTweets = append(selectedTweets, tweets[i])
		}
	}
	if len(selectedTweets) == 0 {
		return nil, errors.New("no tweet at the latest date")
	}

	return selectedTweets, nil
}

// AdjustDate 文字列をtime型にするための補助調整する
func AdjustDate(s string) (time.Time, error) {
	t, err := time.Parse(LAYOUT, s)
	if err != nil {
		return t, err
	}
	// UTC timezone to JST timezone, same time
	t = t.In(time.FixedZone("JST", 9*60*60))
	// 実際の文字列はJSTで保存されているため+9されていない、よって-9でUSTからJSTにした定義分時間を戻す
	t = t.Add(-9 * time.Hour)

	return t, nil
}

// TweetsByChecked チェック有無でTweetsを選択
func TweetsByChecked(account TwitterAccount, tweets []TwitterTweet) ([]TwitterTweet, error) {
	isThere, _, err := Exist(tweets)
	if err != nil || !isThere {
		return nil, err
	}
	// 一つであってもチェックする

	var selectedTweets []TwitterTweet

	for i := 0; i < len(tweets); i++ {
		if tweets[i].Checked == 1 {
			selectedTweets = append(selectedTweets, tweets[i])
		}
	}

	return selectedTweets, nil
}

// SortByPriority PriorityでTweetsを選択 高いものを選択
func SortByPriority(account TwitterAccount, tweets []TwitterTweet) ([]TwitterTweet, error) {
	isThere, l, err := Exist(tweets)
	if err != nil || !isThere {
		return nil, err
	} else if l == 1 { // 選別の必要がない場合
		return tweets, nil
	}

	// Priority属性で降順にソート（つまり、Priorityが高いものが先）
	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].Priority > tweets[j].Priority
	})

	// Priorityが最大のものを選択し、Priority数が変われば終了
	var targetTweets []TwitterTweet
	for i := 0; i < len(tweets); i++ {
		if tweets[i].Priority != tweets[0].Priority {
			break
		}
		targetTweets = append(targetTweets, tweets[i])
	}

	return targetTweets, nil
}

// TweetsByCount CountでTweetsを選択 低いものを選択
// why: 頻出を避けたい
func TweetsByCount(account TwitterAccount, tweets []TwitterTweet) ([]TwitterTweet, error) {
	isThere, l, err := Exist(tweets)
	if err != nil || !isThere {
		return nil, err
	} else if l == 1 { // 選別の必要がない場合
		return tweets, nil
	}

	// Count属性で昇順にソート（つまり、Countが低いものが先）
	sort.Slice(tweets, func(i, j int) bool {
		return tweets[i].Count < tweets[j].Count
	})

	// Countが最小のものを選択し、Count数が変われば終了
	var targetTweet []TwitterTweet
	for i := 0; i < len(tweets); i++ {
		if tweets[i].Count != tweets[0].Count {
			break
		}
		targetTweet = append(targetTweet, tweets[i])
	}

	return targetTweet, nil
}

func Exist(tweets []TwitterTweet) (bool, int, error) {
	if len(tweets) == 0 {
		return false, 0, errors.New("no tweet at check exist")
	}

	return true, len(tweets), nil
}

// SelectTweetsToOne 条件にあったTweetsを選択（現行ランダム
func SelectTweetsToOne(tweets []TwitterTweet) (*TwitterTweet, error) {
	isThere, l, err := Exist(tweets)
	if err != nil || !isThere {
		return nil, fmt.Errorf("%v > no tweet at select one", err)
	} else if l == 1 { // 選別の必要がない場合
		return &tweets[0], nil
	}

	n := rand.Intn(l)

	return &tweets[n], nil
}

func finalCheck(account TwitterAccount, tweet TwitterTweet) error {
	if account.TwitterID != tweet.TwitterID {
		return errors.New("account.twitter_id and tweet.twitter_id are not matched")
	}

	return nil
}
