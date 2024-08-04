package libs

import (
	"fmt"
	"strings"

	"github.com/go-gota/gota/dataframe"

	"github.com/rs/zerolog/log"
)

// GetTwitterAccountList Google spreadsheet「Twitter account list」を取得
// 必ずSheetTitle・RangeKeyを指定すること
// 取得したデータを引数listにBindする
func GetSheet(credByte []byte, spreadID, sheetTitle, rangekey string, bindList any) (dataframe.DataFrame, error) {
	df := dataframe.DataFrame{}

	serv, err := NewSpreadClient(credByte)
	if err != nil {
		return df, SetError(err, "create spread client error")
	}

	// SpreadSheetの情報を取得
	// 概要を取得し、Sheetのタイトルを確認する
	spread, err := serv.Spreadsheets.Get(spreadID).Do()
	if err != nil {
		return df, SetError(err, "get spreadsheet "+spreadID+" error")
	}

	var has_sheet bool
	for _, sheet := range spread.Sheets {
		log.Debug().Msgf("spread has sheet id: %d, sheet title: %s", sheet.Properties.SheetId, sheet.Properties.Title)

		if sheet.Properties.Title == sheetTitle {
			has_sheet = true
		}
	}

	// 指定のSheetTitleがあることを確認してから、データを取得する
	if !has_sheet {
		return df, fmt.Errorf("spreadsheet has no sheet: %s", sheetTitle)
	}

	target := fmt.Sprintf("%s!%s", sheetTitle, rangekey)
	res, err := serv.Spreadsheets.Values.Get(spreadID, target).Do()
	if err != nil {
		return df, SetError(err, "get spreadsheet "+rangekey+" error")
	}

	// 取得したデータを配列に変換する
	rows := make([][]string, len(res.Values))
	for i := 0; i < len(res.Values); i++ {
		var row []string
		for j := 0; j < len(res.Values[i]); j++ {
			row = append(row, strings.TrimSpace(fmt.Sprintf("%v", res.Values[i][j])))
		}
		rows[i] = row
	}
	log.Debug().Msgf("len(rows/row): %d/%d, last row: %s", len(rows), len(rows[len(rows)-1]), rows[len(rows)-1])

	// LoadRecordsで配列をDataframeに読み込む
	df = dataframe.LoadRecords(rows)
	// 指定の型にBingする
	if err := ToStruct(df.Records(), bindList); err != nil {
		return df, SetError(err, "failed to bind list")
	}

	return df, nil
}
