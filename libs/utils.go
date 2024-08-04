package libs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/playwright-community/playwright-go"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"

	ss "google.golang.org/api/sheets/v4"
)

func SetError(err error, msg any) error {
	var s string
	switch v := msg.(type) {
	case string:
		s = v
	case error:
		s = v.Error()
	default:
		s = fmt.Sprintf("%v", msg)
	}

	return fmt.Errorf("%v > %v", err, errors.New(s))
}

// ReadCredentialToByte 認証情報ファイルを読み込み、[]byteにして返す
func ReadCredentialToByte(sfilepath string) []byte {
	f, err := os.Open(sfilepath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open credential file")
	}
	defer f.Close()
	credintial, err := io.ReadAll(f)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read credential file")
	}
	return credintial
}

func NewSpreadClient(cred []byte) (*ss.Service, error) {
	ctx := context.Background()
	var op option.ClientOption
	if len(cred) != 0 {
		op = option.WithCredentialsJSON(cred)
	}
	serv, err := ss.NewService(ctx, op)
	if err != nil {
		return nil, SetError(err, "create client error")
	}

	return serv, nil
}

func Trim(s string) string {
	// Trim "["&"]" from string
	s = strings.TrimLeft(s, "[")
	s = strings.TrimRight(s, "]")
	return s
}

// ToAlphabet 数値をアルファベットに変換する
func ToAlphabet(columnIndex int) string {
	const base = 26
	var columnName string
	for columnIndex > 0 {
		columnIndex-- // 1オフセットの調整
		remainder := columnIndex % base
		columnName = string(rune('A'+remainder)) + columnName
		columnIndex = (columnIndex - remainder) / base
	}
	return columnName
}

// ToStruct Dataframeが出力した[][]stringを指定の型にBindする
func ToStruct(data [][]string, bindList any) error {
	// Reflect on the bindList to verify it's a pointer to a slice of structs.
	bindListVal := reflect.ValueOf(bindList)
	if bindListVal.Kind() != reflect.Ptr || bindListVal.Elem().Kind() != reflect.Slice {
		return errors.New("bindList must be a pointer to a slice of structs")
	}

	// Dig further to verify element type is struct.
	structType := bindListVal.Elem().Type().Elem()
	if structType.Kind() != reflect.Struct {
		return errors.New("bindList must be a pointer to a slice of structs")
	}

	// Assuming first row is headers.
	headers := data[0]

	// Create a map from headers (column names) to struct field indexes.
	fieldMap := make(map[string]int)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("csv")
		for _, h := range headers {
			if h == tag {
				fieldMap[h] = i
				break
			}
		}
	}

	// Iterate over data rows, starting from 1 as we assume 0 is header.
	for _, row := range data[1:] {
		newStructPtr := reflect.New(structType).Elem()
		for col, value := range row {
			fieldIndex, exists := fieldMap[headers[col]]
			if !exists {
				continue // Skip if no matching struct field
			}
			fieldVal := newStructPtr.Field(fieldIndex)

			// Handle basic data types - expand as necessary.
			switch fieldVal.Kind() {
			case reflect.String:
				fieldVal.SetString(value)
			case reflect.Int, reflect.Int32, reflect.Int64:
				if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
					fieldVal.SetInt(intValue)
				}
				// Add more type handlers as necessary.
			}
		}
		bindListVal.Elem().Set(reflect.Append(bindListVal.Elem(), newStructPtr))
	}
	return nil
}

func newPage(is_post bool) (*playwright.Playwright, playwright.Browser, playwright.Page, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, nil, nil, SetError(err, "could not run playwright")
	}

	// is_post = falseならば、GUIブラウザを表示して操作する
	browser, err := pw.Firefox.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(is_post),
	})
	if err != nil {
		return nil, nil, nil, SetError(err, "could not launch browser")
	}

	// for key, device := range pw.Devices {
	// 	fmt.Printf("%v, %v\n", key, device)
	// }

	// モバイルデバイスの設定
	// 既知のバグを回避するため、旧機種Pixel 5を使用
	// 日本東京の緯度経度を指定
	device := pw.Devices["iPad Pro 11 landscape"]
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		AcceptDownloads:   playwright.Bool(true),
		DeviceScaleFactor: playwright.Float(device.DeviceScaleFactor),
		Geolocation:       &playwright.Geolocation{Longitude: 139.749281, Latitude: 35.6959983},
		HasTouch:          playwright.Bool(device.HasTouch),
		// IsMobile:          playwright.Bool(device.IsMobile),
		JavaScriptEnabled: playwright.Bool(true),
		Locale:            playwright.String("ja-JP"),
		// Permissions:       []string{"geolocation", "background-sync"},
		RecordHarContent: playwright.HarContentPolicyAttach,
		TimezoneId:       playwright.String("Asia/Tokyo"),
		ServiceWorkers:   playwright.ServiceWorkerPolicyAllow,
		UserAgent:        playwright.String(device.UserAgent),
		Viewport:         device.Viewport,
	})
	if err != nil {
		return nil, nil, nil, SetError(err, "could not create device context")
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, nil, nil, SetError(err, "could not create new page")
	}

	return pw, browser, page, nil
}

func pwClose(pw *playwright.Playwright, page playwright.Page) {
	page.Close()
	pw.Stop()
}

func millisec(r *rand.Rand) int {
	return r.Intn(2000) + 1000
}
