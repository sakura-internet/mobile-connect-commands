package main

import (
	"encoding/csv"
	"fmt"
	"strings"

	"os"
	"testing"

	"github.com/sakura-internet/mobile-connect-commands/common"
)

/***
 * テスト実行前にtestdataディレクトリに以下のファイルを配置しておくこと
 * - テスト設定ファイル config.csv
 *   CSVの形式は
 *    アクセストークン, アクセストークンシークレット, 対象のモバイルゲートウェイのゾーン名, 対象のモバイルゲートウェイのリソースID
 *   で、最初の1行のみ有効
 * - 形式CSVファイル simlist.csv
 ***/

/*
 * SIM登録テストに必要な設定をtestdata/config.csvから読み込む
 */
type RegisterSimTestConfig struct {
	AccessToken       string
	AccessTokenSecret string
	Zone              string
	MgwId             string
}

// 設定ファイル読み込み
func readTestConfig() (RegisterSimTestConfig, error) {
	csvPath := "testdata/config.csv"

	// CSVファイルを開く
	file, err := os.Open(csvPath)
	if err != nil {
		return RegisterSimTestConfig{}, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	record, err := reader.Read()
	if err != nil {
		return RegisterSimTestConfig{}, err
	}

	if len(record) < 4 {
		return RegisterSimTestConfig{}, fmt.Errorf("CSVのレコードが足りません...record: %v", record)
	}

	return RegisterSimTestConfig{AccessToken: record[0], AccessTokenSecret: record[1], Zone: record[2], MgwId: record[3]}, nil
}

var testConfig RegisterSimTestConfig

func init() {
	// テストに使用する設定を読み込む
	var err error
	testConfig, err = readTestConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}

	err = validateZone(testConfig.Zone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}
}

/** テストコード **/

func TestLoadSimListCsv(t *testing.T) {
	t.Run("CSVファイルを読み込みSIMリストを返す", func(t *testing.T) {
		csvPath := "testdata/load_test.csv"

		// CSVファイルを読み込む
		simList, err := loadSimListCsv(csvPath)
		if err != nil {
			t.Fatalf("CSVファイルが読み込めません。%s", err.Error())
		}

		for _, sim := range simList {
			t.Logf("ICCID: %s, PassCode: %s\n", sim.ICCID, sim.PassCode)
		}

		t.Log("OK")
	})
}

func TestRegisterSimFromList(t *testing.T) {
	t.Run("使用できるIPアドレスが不足している場合エラーになる", func(t *testing.T) {
		// SIMのリスト
		simList := []common.SimRegisterInfo{
			{ICCID: "1234560000000123400", PassCode: "RanDOmPasS"},
			{ICCID: "1234560000000123401", PassCode: "rANdoMpAAs"},
		}
		// IPのアドレスのリスト
		ipAddrs := []string{
			"172.31.30.1",
		}

		// SIM登録実行(IPアドレスが足りないのでエラーが発生するはず)
		err := common.RegisterSimFromList(testConfig.AccessToken, testConfig.AccessTokenSecret, testConfig.Zone, testConfig.MgwId, simList, ipAddrs)
		if err != nil {
			if strings.HasPrefix(err.Error(), "登録対象のSIM") {
				t.Log("OK")
				return
			}
			t.Fatalf("想定外のエラーが発生しました...%s", err.Error())
		}
		t.Fatalf("失敗するはずが成功してしまいました")
	})

	t.Run("SIM登録を実行する", func(t *testing.T) {
		/**
		 * 事前に有効なCSVファイルをtestdata/simlist.csvとしておいておく
		 */
		t.Logf("Config: %v", testConfig)
		// SIMのリストを読み込む
		csvPath := "testdata/simlist.csv"
		simList, err := loadSimListCsv(csvPath)
		if err != nil {
			t.Fatalf("ICCID,passcodeの形式で登録可能なSIMが列挙されているCSVファイルが必要です:  %s", err.Error())
		}
		t.Logf("simList: %v", simList)

		// IPアドレスのリスト(ダミーで253個生成)
		var ipAddrs []string
		for i := 1; i < 255; i++ {
			ipAddrs = append(ipAddrs, fmt.Sprintf("172.31.30.%d", i))
		}

		// SIM登録
		err = common.RegisterSimFromList(testConfig.AccessToken, testConfig.AccessTokenSecret, testConfig.Zone, testConfig.MgwId, simList, ipAddrs)
		if err != nil {
			t.Fatalf("%s", err.Error())
		}
	})
}

func TestValidateCIDR(t *testing.T) {
	t.Run("不正なCIDRを入力したらエラーが返る", func(t *testing.T) {
		cidr := "192.168.1.0.0/29"
		_, _, err := validateCIDR(cidr)

		if err != nil {
			t.Log("OK")
		} else {
			// エラーが返ることが正しいので、こちらはおかしい
			t.Fatalf("error is expected to be returned, but nothing is returned.\n")
		}
	})

	t.Run("正常なCIDRはエラーが返らない", func(t *testing.T) {
		cidr := "192.168.1.0/29"
		_, _, err := validateCIDR(cidr)

		if err != nil {
			t.Fatalf("nil error is expected, but got %s", err.Error())
		} else {
			t.Log("OK")
		}
	})
}
func TestValidateZone(t *testing.T) {
	t.Run("不正なゾーンを入力したら、エラーが返る", func(t *testing.T) {
		zone := "tk3a"
		err := validateZone(zone)
		if err != nil {
			t.Log("OK")
		} else {
			t.Fatalf("error is expected")
		}
	})
}
func TestValidateArgs(t *testing.T) {
	t.Run("アクセストークン、アクセストークンシークレットが無いとエラーになる", func(t *testing.T) {
		options := Options{CsvPath: "testdata.csv", AccessToken: "", AccessTokenSecret: "", Zone: "is1a", CIDR: "192.168.1.0/29", MgwResourceID: "aaaaaaa"}
		_, _, err := validateArgs(options)
		if err != nil {
			t.Log("OK")
		} else {
			t.Fatalf("error is expected")
		}
	})

	t.Run("CSVファイルのパスが無いとエラーになる", func(t *testing.T) {
		options := Options{CsvPath: "", AccessToken: "Token", AccessTokenSecret: "Secret", Zone: "is1a", CIDR: "192.168.1.0/29", MgwResourceID: "aaaaaaa"}
		_, _, err := validateArgs(options)
		if err != nil {
			t.Log("OK")
		} else {
			t.Fatalf("error is expected")
		}
	})
}
