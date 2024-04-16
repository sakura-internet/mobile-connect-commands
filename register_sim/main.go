package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"slices"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"github.com/sakura-internet/mobile-connect-commands/common"
)

// コマンドライン引数
type Options struct {
	CsvPath           string `long:"csv" description:"CSVファイルのパス"`
	AccessToken       string `long:"token" description:"さくらのクラウドAPIアクセストークン"`
	AccessTokenSecret string `long:"secret" description:"さくらのクラウドAPIアクセスシークレット"`
	Zone              string `long:"zone" description:"さくらのクラウドゾーン"`
	CIDR              string `long:"cidr" description:"探索対象のCIDR"`
	MgwResourceID     string `long:"mgw-resource-id" description:"モバイルゲートウェイのリソースID"`
}

// validateZone
// 正しい Zone かチェックする
func validateZone(zone string) error {
	validZones := []string{"tk1a", "tk1b", "is1a", "is1b"}
	if !slices.Contains(validZones, zone) {
		return fmt.Errorf("不正なゾーンです。%s から指定してください", strings.Join(validZones, ", "))
	}
	return nil
}

func validateCIDR(cidr string) (net.IP, *net.IPNet, error) {
	// CIDR のパースだけ行う
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ip, ipNet, fmt.Errorf("正しいフォーマットのCIDRを指定してください: %s", err.Error())
	}
	return ip, ipNet, nil
}

// コマンドライン引数のバリデーションを行う
func validateArgs(opts Options) (net.IP, *net.IPNet, error) {
	if opts.CsvPath == "" {
		return nil, nil, errors.New("コマンドライン引数にCSVファイルのパスを指定してください")
	}

	if opts.MgwResourceID == "" {
		return nil, nil, errors.New("コマンドライン引数にモバイルゲートウェイのリソースIDを指定してください")
	}

	if (opts.AccessToken == "") || (opts.AccessTokenSecret == "") {
		return nil, nil, errors.New("コマンドライン引数にAPIアクセストークンとAPIアクセストークンシークレットを指定してください")
	}

	err := validateZone(opts.Zone)
	if err != nil {
		return nil, nil, err
	}

	ip, ipNet, err := validateCIDR(opts.CIDR)
	if err != nil {
		return nil, nil, err
	}

	return ip, ipNet, nil
}

func loadSimListCsv(csvPath string) ([]common.SimRegisterInfo, error) {

	sim := make([]common.SimRegisterInfo, 0, 100)

	// CSVファイルを開く
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("CSVファイルのオープンに失敗しました...%s", err.Error())
	}
	defer file.Close()

	// ICCIDをキーにパスコードを追加
	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// ファイルの末尾に到達
				break
			} else {
				var parseErr *csv.ParseError
				if !errors.As(err, &parseErr) {
					// その他のエラー
					return nil, fmt.Errorf("読み込みに失敗しました...%s", err.Error())
				}
			}
		}
		if len(record) != 2 {
			// フィールド数が一致しない
			lineNo, _ := reader.FieldPos(0)
			return nil, fmt.Errorf("%d行目:列数が正しくありません...2列必要ですが%d列読み込みました", lineNo, len(record))
		}
		sim = append(sim, common.SimRegisterInfo{ICCID: record[0], PassCode: record[1]})
	}

	return sim, nil
}

func main() {
	// コマンドライン引数の確認
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "コマンドライン引数のパースに失敗しました...%s\n", err.Error())
		os.Exit(1)
	}

	ip, ipNet, err := validateArgs(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "コマンドライン引数が不正です...%s\n", err.Error())
		os.Exit(1)
	}

	// CSVの読み込み
	fmt.Printf("CSVファイル(%s)の読み込み中...", opts.CsvPath)
	sim, err := loadSimListCsv(opts.CsvPath)
	if err != nil {
		// エラーメッセージを出力
		fmt.Println("[NG]")
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	fmt.Println("[OK]")

	// MGWで使用中のIPアドレスのリストを取得
	fmt.Printf("使用可能なIPアドレスの取得中...")
	mgwIPAddrs, err := common.GetUsedIPAddressesInMGW(opts.AccessToken, opts.AccessTokenSecret, opts.Zone, opts.MgwResourceID)
	if err != nil {
		// エラーメッセージを出力
		fmt.Println("[NG]")
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	// 使用可能なIPアドレスのリストを取得
	availableIPAddrs := make([]string, 0, len(mgwIPAddrs))
	for ipaddr := range common.GetAvailableIPAddresses(ip, ipNet, mgwIPAddrs) {
		availableIPAddrs = append(availableIPAddrs, ipaddr)
	}
	fmt.Println("[OK]")

	// SIMを登録
	fmt.Println("SIM一括登録 開始")
	err = common.RegisterSimFromList(opts.AccessToken, opts.AccessTokenSecret, opts.Zone, opts.MgwResourceID, sim, availableIPAddrs)
	if err != nil {
		// 登録に失敗
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("SIM一括登録 完了")

	os.Exit(0)
}
