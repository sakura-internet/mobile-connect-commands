package main

import (
	"errors"
	"fmt"
	flags "github.com/jessevdk/go-flags"
	"github.com/sakura-internet/mobile-connect-commands/common"
	"golang.org/x/exp/slices"
	"net"
	"os"
	"strings"
)

// コマンドライン引数
type Options struct {
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

func main() {
	// コマンドラインオプションのパース
	var opts Options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()

	if err != nil {
		fmt.Fprintf(os.Stderr, "コマンドライン引数のパースに失敗しました...%s\n", err.Error())
		os.Exit(1)
	}

	// コマンドライン引数を バリデーションする
	ip, ipNet, err := validateArgs(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "コマンドライン引数が不正です...%s\n", err.Error())
		os.Exit(1)
	}

	// コマンドラインオプションのチェックが終わったら、実行していることを
	// ユーザに伝えるため情報を出す
	fmt.Println("情報を取得しています...")

	mgwIPAddrs, err := common.GetUsedIPAddressesInMGW(opts.AccessToken, opts.AccessTokenSecret,
		opts.Zone, opts.MgwResourceID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	for ipaddr := range common.GetAvailableIPAddresses(ip, ipNet, mgwIPAddrs) {
		// 取得可能なIPアドレスを表示する
		fmt.Println(ipaddr)
	}
	os.Exit(0)
}
