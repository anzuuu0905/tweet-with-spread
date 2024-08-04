/*
Rules:

- プロジェクトにより変化するデータ構造や条件などは当ファイル内関数で設定することで汎化性を確保する
- Google Cloudクレデンシャルファイルを取得し指定ファイルパスに存在すること -> Google spreadsheetにアクセス権を得る
- Google Spreadsheet APIが有効であること -> Google spreadsheetにアクセスする権限をアカウント及びクレデンシャルに付与する
- Google spreadsheetはURL共有状態であること -> Google spreadsheetにアクセスされることを承認する
- Google spreadsheet Sheet各項目に必要情報が記載されていること -> 項目を指定して読み込み、処理を行うために整理する
- Google spreadsheet各項目は数字であれ文字列（表示形式はいじらない）とすること -> プログラムで文字列を数値にする
- Google spreadsheet各項目でYes/Noを表現する場合は半角数字0/1であること -> プログラムで文字列の0/1をBool型にし、1であればYes、その他数字はNoとする
- Google spreadsheetで[files]はセル区切りで4つまで記述可能。文字列は半角英数字・Spaceなし -> プログラムでセル区切りの文字列を配列にする。重いファイルは無視される
- Google spreadsheetでFile各項は同アカウント内Dirveに保存されたFileであり、FileID及びFileIDを含むURLであること -> プログラムで文字列を取得しダウンロード、Fileデータを生成する
- Google spreadsheetでhours, minutesは半角数字で、,区切りで指定する -> プログラムで文字列を数値の配列にする
- Google spreadsheetでプログラムによって更新される列はかならず最後の列であること -> プログラムで最後の列を指定し更新する
- Google spreadsheetで年月日指定は半角数字記号でYYYY/MM/DD HH:MM:SSであること -> プログラムで年月日を指定し、日付を比較する


*/

package main

import (
	"os"
	"time"
	"tweet-with-spread/cmd/User596E9F4/subsets"
	"tweet-with-spread/libs"

	"github.com/go-gota/gota/dataframe"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	// Debug: プログラムの実行環境
	IS_PRODUCT = true
	// Debug: Twitter投稿を行うかどうか
	IS_TWITTER_POST = true
	// 起動間隔
	INTERVAL time.Duration = 1 * time.Minute

	// 文字数での投稿先の分岐 -> GUI/API
	TWEETCOUNT_JA = 140

	// Google Spreadsheet Accounts Sheet ID
	ACCOUNTSHEETTITLE = "twitter_users"
	// Google Spreadsheet Tweets Sheet ID
	TWEETSSHEETTITLE = "twitter_tweets"
	// Google Spreadsheet Search Sheet ID
	SEARCHSHEETTITLE = "twitter_search"
	// Google Spreadsheet Sheetの取得範囲
	SHEET_RANGE = "A1:Z"

	// ランダム待機時間（秒）
	MAXWAITSEC = 150
)

var (
	// Google Cloudクレデンシャルファイル
	// 同階層に置いてください
	CREDENTIALJSONFILE string

	// Google Spreadsheet ID
	// AccountsListが記載されている管理者用Spreadsheet
	SPREADSHEET_ID string
)

func init() {
	// ログの設定
	// 出力レベルを変える
	// 	- Debug: デバッグ用
	// 	- Info: 通常のログ
	// 	- Warn: 警告
	// 	- Error: エラー
	// 	- Fatal: 致命的なエラー
	// 	- Panic: プログラムが続行できないエラー
	if IS_PRODUCT {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// ファイル場所の指定
	CREDENTIALJSONFILE = os.Getenv("CREDENTIAL")
	if CREDENTIALJSONFILE == "" {
		log.Fatal().Msg("CREDENTIAL is not set")
	}

	// Google Spread IDの指定
	SPREADSHEET_ID = os.Getenv("SPREADSHEET_ID")
	if SPREADSHEET_ID == "" {
		log.Fatal().Msg("SPREADSHEET_ID is not set")
	}
}

func main() {
	// INTERVAL値による定期実行
	t := time.NewTicker(INTERVAL)
	defer t.Stop()

	// 認証があるSpreadsheetを取得する場合はCREDENTIALJSONFILEを指定する
	// ない場合は起動せずFatalで終了する
	cred := libs.ReadCredentialToByte(CREDENTIALJSONFILE)

	// TwitterAPIのHTTPリクエストをインターセプトする
	// ‐ API Limitを取得し、残り回数でリクエストを制御する
	li := libs.NewLoggingInterceptor()

	// 分の開始0秒に開始するために、初回の実行を待つ
	sub := time.Since(time.Now().Truncate(time.Minute))
	time.Sleep(INTERVAL - sub)

	log.Info().Msg("Start Program")

	for t := range t.C {
		// 定期実行本体: 並列処理/待機有り
		// ‐ INTERVAL毎にExecutorを実行する
		// ‐ Executor内でSpreadsheetからデータを取得し、Tweetを投稿する
		// ‐ Tweet投稿後、Spreadsheetに投稿日を記録する
		// ‐ その後、次のINTERVALを待つ

		// Executor内のエラーについて
		// ‐ Executor内で実行するための必要なファイルが見つからない場合は、Fatalでメインプログラムごと強制終了
		// ‐ Executor内でエラーが発生した場合、ログを出力し終了
		// - 次のINTERVALを待つ
		// - 実行関数が呼び出された時間を引数にする
		if t.Minute()%5 != 0 {
			continue
		}
		go Executor(cred, li, t)

		// 1日の終りに一時ファイルを削除
		if t.Hour() == 0 && t.Minute() == 0 {
			if err := libs.CleanDir(subsets.TEMPORARYDIR); err != nil {
				log.Err(err).Msg("failed to remove files")
			}
		}
	}
}

// Executor Google Scheduleで定期実行することを想定
// Pingが飛んできたら実行する
func Executor(cred []byte, li *libs.LoggingInterceptor, t time.Time) {
	log.Info().Str("function", "Executor").Msg("start")
	// Google spreadsheet「Twitter account list」を取得、指定の方にBindする
	// 追加でDataframeを返しているが、現行未使用
	// ‐ 適用案: Twitter account listの行に「0/1」を含む列を作り、投稿の可否を管理する
	twitterAccounts := make([]subsets.TwitterAccount, 0)
	_, err := libs.GetSheet(cred, SPREADSHEET_ID, ACCOUNTSHEETTITLE, SHEET_RANGE, &twitterAccounts)
	if err != nil {
		log.Error().Err(err).Str("function", "Executor").Msgf("Failed to get list")
		return
	}

	// Twitter account listから投稿するべきアカウントを取得する
	targetAccounts, err := subsets.SelectTwitterAccounts(t, twitterAccounts)
	if err != nil {
		log.Debug().Str("function", "Executor").Msgf("%v > no accounts in list", err)
		return
	}

	// // For内において１アカウント、１Tweet
	// 現状、アカウント別のツイート選択条件はない。同一条件である
	for i := 0; i < len(targetAccounts); i++ {
		// Google spreadsheet「Tweets list for Target Account」を取得
		// ‐ 現行: 同Spreadsheet内Sheet
		// ‐ 適用案: Twitter account list各IDに該当するSpreadsheetIDとSheetID項目を付与する
		twitterTweets := make([]subsets.TwitterTweet, 0)
		dfTweets, err := libs.GetSheet(cred, targetAccounts[i].SpreadID, TWEETSSHEETTITLE, SHEET_RANGE, &twitterTweets)
		if err != nil {
			log.Error().Err(err).Str("function", "Executor").Msgf("failed to get list, %s", targetAccounts[i].TwitterID)
			continue
		}

		// Tweetsから指定条件で抜粋
		tweet, err := subsets.SelectTweet(targetAccounts[i], twitterTweets)
		if err != nil {
			log.Error().Err(err).Str("function", "Executor").Msgf("filed select tweets, %s", targetAccounts[i].TwitterID)
			continue
		}

		// // Debug: Tweetを指定
		// tweet = &twitterTweets[len(twitterTweets)-1]
		if !IS_PRODUCT {
			log.Debug().Msgf("all rows start-----------------")
			for i := 0; i < len(targetAccounts); i++ {
				log.Debug().Str("function", "Executor").Msgf("account: %+v", targetAccounts[i])
			}

			for i := 0; i < len(twitterTweets); i++ {
				log.Debug().Str("function", "Executor").Msgf("tweets: %+v", twitterTweets[i])
			}

			log.Debug().Str("function", "Executor").Msgf("select one: %+v", tweet)

			log.Debug().Str("function", "Executor").Msgf("all rows end-----------------")

			return
		}

		// ランダムな待機時間を設定
		subsets.Wait(MAXWAITSEC)

		// 長文ツイートでの分岐
		// 	// Option: 選択したTweetに画像が含まれる場合は画像をアップロードしてMediaIDを取得する
		log.Debug().Str("function", "Executor").Msgf("selected tweet id: %+v", tweet.Index)
		var files = tweet.Tofiles(cred)
		log.Debug().Str("function", "Executor").Msgf("setup files: %+v", files)
		if len([]rune(tweet.Text)) > TWEETCOUNT_JA {
			if err := libs.TweetsToGUI(
				IS_TWITTER_POST,
				tweet.WithFiles == 1,
				targetAccounts[i].TwitterID,
				targetAccounts[i].Password,
				tweet.Text,
				files); err != nil {
				log.Err(err).Msgf("failed to tweeting for GUI, %s: %d", targetAccounts[i].TwitterID, tweet.Index)
				continue
			}

			// 要検討: GUIで投稿した場合は、TwitterAPI v1 TweetsでTweetURLを更新
			// API Limitを消費するため、現状は未実装

		} else {
			req, err := subsets.RequestCreateTweet(targetAccounts[i], tweet, files)
			if err != nil {
				log.Error().Err(err).Str("function", "Executor").Msgf("failed to create tweet request, %s: %d", targetAccounts[i].TwitterID, tweet.Index)
				continue
			}
			log.Debug().Str("function", "Executor").Msgf("tweet request: %+v", req)
			res, err := li.Tweeting(
				IS_TWITTER_POST,
				targetAccounts[i],
				req,
			)
			if err != nil {
				log.Error().Err(err).Str("function", "Executor").Msgf("failed to tweeting, twitter id: %s, index: %d", targetAccounts[i].TwitterID, tweet.Index)
				continue
			}
			// TweetURLを更新
			tweet.TweetURL = libs.ID2TwitterURL(*res.Data.ID)
			log.Info().Str("function", "Executor").Msgf("success tweeted: %s", tweet.TweetURL)
		}

		// 行の更新にかかる変更処理はここで行う
		// Dataframeに対して、行・列を指定して更新を行い、UpdateRowでSpreadsheetに反映する
		// ## 現行:
		// - Countの更新
		// - TweetURLの更新
		// - 最終投稿日の更新

		// 投稿したTweetsをGoogle spreadsheet「Tweets list」に保存（現行は指定セルである「投稿日」の上書き
		_, colN := dfTweets.Dims()
		dfTweets.Elem(tweet.Index-1, colN-3).Set(tweet.Count + 1)
		dfTweets.Elem(tweet.Index-1, colN-2).Set(tweet.TweetURL)
		dfTweets.Elem(tweet.Index-1, colN-1).Set(time.Now().Format(subsets.LAYOUT))

		// 行を指定しUpdateRequestの指定する型に整形する
		targetRangeKey, row, err := libs.SubsetToUpdateRowWithRangeKey(dfTweets, tweet.Index, TWEETSSHEETTITLE)
		if err != nil {
			log.Err(err).Msg("failed to subset to row with range key")
		}
		// 行を更新する
		if err := libs.UpdateRow(cred, SPREADSHEET_ID, TWEETSSHEETTITLE, targetRangeKey, row); err != nil {
			log.Warn().Msgf("failed to update cell: %s", err)
		}

	} // end of for
}

// UpdateDataframe Dataframeを更新する
func UpdateDataframe(df dataframe.DataFrame, tweet subsets.TwitterTweet) (bool, int, dataframe.DataFrame) {
	var (
		isChanged bool
		rowN      int
	)

	rN, cN := df.Dims()
	for i := 0; i < rN; i++ {
		// Indexが一番左の列にあると見込む
		index, err := df.Elem(i, 0).Int()
		if err != nil {
			continue
		}

		if index == tweet.Index {
			// 行を変更していなければtweet.Indexがある場所は、rowN+1となる
			// しかし、並べ替えやSortしているとIndexが降順であるとは限らない
			// そのため、投稿したIndexと一致する行を探し、その行を更新する
			// Spreadsheetを取得してから、投稿するまでの間に行が変更されたりする場合はサポート外
			if tweet.Index != i {
				isChanged = true
			}
			rowN = i
			df.Elem(rowN, cN-3).Set(tweet.Count + 1)
			df.Elem(rowN, cN-2).Set(tweet.TweetURL)
			df.Elem(rowN, cN-1).Set(time.Now().Format(subsets.LAYOUT))
			break
		}
	}

	return isChanged, rowN, df
}
