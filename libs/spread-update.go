package libs

import (
	"errors"
	"fmt"

	"github.com/go-gota/gota/dataframe"
	"github.com/rs/zerolog/log"
	ss "google.golang.org/api/sheets/v4"
)

// UpdateRow Google spreadsheetの指定行を更新する
// 必ずSheetTitle・RangeKeyを指定すること
// SubsetToUpdateRowWithRangeKeyを使い、RangeKeyとRowを抽出するこことを推奨
func UpdateRow(credByte []byte, spreadID, sheetTitle, rangeKey string, row []interface{}) error {
	serv, err := NewSpreadClient(credByte)
	if err != nil {
		return SetError(err, errors.New("create spread client error"))
	}

	// 値の更新、該当行を上書きする
	if _, err := serv.Spreadsheets.Values.Update(spreadID, rangeKey, &ss.ValueRange{
		MajorDimension: "ROWS",
		Values: [][]interface{}{
			row,
		},
	}).ValueInputOption("USER_ENTERED").Do(); err != nil {
		return SetError(err, errors.New("update spreadsheet "+rangeKey+" error"))
	}

	return nil
}

// SubsetToUpdateRowWithRangeKey DataFrameの指定行を更新する
// ‐ convertToUpdateRequestRowsWithHeader
func SubsetToUpdateRowWithRangeKey(df dataframe.DataFrame, updateIndex int, sheetTitle string) (rangeKey string, row []interface{}, err error) {
	// SpreadSheetの情報を取得し
	// データに差異がある部分を上書きする
	keyAndRow := df.Subset([]int{updateIndex - 1})
	records := keyAndRow.Records()

	// Spreadsheetが受けつける型に変換する
	in := convertToUpdateRequestRowsWithHeader(records)
	if len(in) == 0 {
		return "", nil, SetError(errors.New("no data"), "convert for update rows by request type")
	}

	// Updateする列は最後の列である
	_, colN := df.Dims()
	// 指定行数の列先端から末尾までを更新
	// Reason: 行先頭にはColumn_Nameが入っているため
	spreadUpdateIndex := updateIndex + 1
	target := fmt.Sprintf("%s!A%d:%s%d", sheetTitle, spreadUpdateIndex, ToAlphabet(colN), spreadUpdateIndex)
	log.Debug().Msgf("target range key: %s, update row: %v", target, in[1])

	return target, in[1], nil
}

func convertToUpdateRequestRowsWithHeader(records [][]string) [][]interface{} {
	in := make([][]interface{}, len(records))
	for i := 0; i < len(records); i++ {
		in[i] = make([]interface{}, len(records[i]))
		for j, cell := range records[i] {
			// Important!! []string -> []interface{}
			in[i][j] = fmt.Sprintf("%v", Trim(cell))
		}
	}
	return in
}
