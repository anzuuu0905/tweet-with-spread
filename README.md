---
marp: true
---

# Tweet With Spread

このプロジェクトは、**Google Spreadsheet**の情報を使用して**Twitter**に自動で投稿するためのアプリケーションです。設定された間隔ごとに特定のSpreadsheetからデータを読み込み、Twitterに投稿します。

---

# 開発者ドキュメント
当ソースコードに記載した開発用ドキュメントとして関数（処理）ごとに説明文章を付与しました。  
以下コマンドを使用することでHTMLを出力し、ブラウザで見ることができます。  
説明がないものは親としての処理を補助するものです。  


```sh
$ godoc -http=localhost:8080
```
[godoc](http://localhost:8080/pkg/tweet-with-spread/)
---

---

## 機能

- **Twitterアカウントの管理:** 複数のTwitterアカウント情報をGoogle Spreadsheetから取得し、それらを利用して投稿を行います。
- **ツイート情報の管理:** 投稿するツイートの内容をGoogle Spreadsheetから取得し。画像ファイルの指定やツイートの優先度などもSpreadsheetから設定できます。
- **自動投稿:** 当プログラムアプリケーションは設定された間隔(`INTERVAL`)ごとにSpreadsheetから投稿データを取得し、Twitterに自動投稿します。
- **長文投稿 for Blue(Pro)** GUIを使用し、長文投稿を行います。現在、画像・動画アップロードをサポート。サイズや形式により、エラーの可能性があります。Twitter/X Documentを参照ください。
- **投稿選択** 日時・他項目で投稿候補を選別します。選別条件の追記・変更などに関しては実装関数を分離しています、詳細はSelect***関連の関数を参照ください。
- **ゆらぎ(乱数待機)** 定期実行関数が実行され諸処理が終了次第、投稿前に指定時間以下で乱数で待機時間を設けます。並列処理が可能です、ゆらぎ待機中でも次の実行が行われます。
- **投稿ログ:** 投稿の回数、URL、日時ログ情報を通して実行結果を確認することができます。GUI投稿の場合はURLを取得しません。
- **エラーハンドリング:** 不足しているデータやファイルがある場合、エラーをログとして記録し、投稿をスキップします。

---

## 使用上のルール

- Google Cloudのクレデンシャルと対応するURL共有済みのSpreadsheetが必須です。
- 使用するTwitterアカウントの認証情報(`Consumer Key`, `Consumer Secret`, `Access Token`, `Secret Token`)が必要です。
- 投稿するツイートは、指定されたフォーマットでSpreadsheetに保存してください。
- 定期実行には、サーバーまたはスケジューラが必要です。(例: GCE, Google Functions/Run, Google Cloud Scheduler)

---

## Spreadsheet記述及びプログラムの仕様・ルール
- post用text以外は半角英数字記号であること
- プロジェクトにより変化するデータ構造や条件などは当ファイル内関数で設定することで汎化性を確保する
- 上記の上で、列名・行列の並びは変えない。プログラムがindexを参照しても、行番号を参照しても、Spreadデータを取得してから書き込むまでに時間が生じるためバグの原因をルールで制限。
- Google Cloudクレデンシャルファイルを取得し指定ファイルパスに存在すること -> Google spreadsheetにアクセス権を得る。プログラム側で保持・設定済み
- Google Spreadsheet APIが有効であること -> Google spreadsheetにアクセスする権限をアカウント及びクレデンシャルに付与する。設定済み
- Google spreadsheet・参照ファイル群はURL共有状態であること -> Google spreadsheet API及びプログラムからアクセスされることを承認するため
---

- Google spreadsheet Sheet各項目に必要情報が記載されていること -> 項目を指定して読み込み、処理を行うために整理する
- Google spreadsheet各項目は数字であれ文字列（表示形式はいじらない）とすること -> プログラムで文字列を数値にする
- Google spreadsheet各項目でYes/Noを表現する場合は半角数字0/1であること -> プログラムで文字列の0/1をBool型にし、1であればYes、その他数字はNoとする
- Google spreadsheetで[files]はセル区切りで4つまで記述可能。文字列は半角英数字・Spaceなし -> プログラムでセル区切りの文字列を配列にする。サポートファイル:"image/jpeg,image/png,image/webp,image/gif,video/mp4,video/quicktime"、重いファイルは無視される（要with_filesで指定してください）
---

- Google spreadsheetでFile各項は同アカウント内Driveに保存されたFileであり、FileID及びFileIDを含むURLであること -> プログラムで文字列を取得しダウンロード、Fileデータを生成する。※同様の画像及び動画がTwitter上で投稿履歴があるときエラーになる。
- Google spreadsheetでhours, minutesは半角数字で、[,]区切りで指定する -> プログラムで半角数字と[,]文字列を数値の配列にする
- Google spreadsheetでプログラムによって更新される列はかならず最後の列であること -> プログラムで最後の列を指定し更新する
- Google spreadsheetで年月日指定は半角数字記号でYYYY/MM/DD HH:MM:SSであること -> プログラムで年月日を指定し、日付を比較する
---

## 使用方法

1. 必要なクレデンシャルファイルとSpreadsheetのIDを設定する。
2. Twitterアカウントの認証情報とツイート情報をSpreadsheetに入力する。
3. アプリケーションをデプロイし、指定された間隔で実行されるよう設定する。
4. 投稿されたツイートやログを確認する。

---

## 定数
- `IS_PRODUCT`: プロダクションモード、ログの出力レベル
- `IS_TWITTER_POST`: テストモードか実際にTwitterに投稿するかのフラグ。`false`の場合は投稿せず、ログのみ表示します。
- `INTERVAL`: 起動を行う間隔。毎分起動し、分針が5の倍数であれば実行します。
- `TWEETCOUNT_JA`: 日本語ツイートの文字数制限。超えた場合はGUIへの投稿となります。
- `CREDENTIALJSONFILE`: Google Cloudのクレデンシャルファイルへのパス。
- `SPREADSHEET_ID`: Twitterアカウントとツイート情報を管理しているGoogle SpreadsheetのID。
- 各Sheetのタイトル(`ACCOUNTSHEETTITLE`, `TWEETSSHEETTITLE`, `SEARCHSHEETTITLE`): 対応するデータを管理するSheetの名前。
- `TEMPORARYDIR`: 一時ファイルを保存するディレクトリへのパス。
-	`MAXWAITSEC`: ゆらぎ、投稿までのランダム待機時間（秒）, default: 150

開発者用定数:
- `MAXWAITFORUPLOAD`: GUI用 ファイルアップロードまでの最大待機時間。インスタンスや頻出ファイルなどにより適宜変更。default: 120（秒）
---

## 注意点

- 本アプリケーションは、Google Cloud上にて適切なクレデンシャルとAPIの利用同意が必要です。
- TwitterのAPI使用制限に注意してください。特に画像のアップロードや大量のツイート投稿は、制限に引っ掛かることがあります。
- 設定や操作を間違えると、予期しないアカウントでの投稿や不適切な内容の投稿となることがあります。利用には十分注意してください。
----------------