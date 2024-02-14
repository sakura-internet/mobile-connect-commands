package main

import (
	"github.com/sakura-internet/mobile-connect-commands/common"
	"net"
	"reflect"
	"testing"
)

func TestAvailableIPAddress(t *testing.T) {
	t.Run("モバイルゲートウェイの使用済みIPアドレスから利用可能なIPアドレスを返す", func(t *testing.T) {
		cidr := "192.168.1.0/29"
		ip, ipNet, _ := net.ParseCIDR(cidr)
		blank := struct{}{}

		// SIM にアサイン済の IP アドレス
		simAssignedIPAddresses := map[string]struct{}{
			"192.168.1.1": blank,
			"192.168.1.2": blank,
			"192.168.1.3": blank,
			"192.168.1.4": blank,
		}

		// ネットワークアドレスとブロードキャストアドレスは除外される
		assertResult := map[string]struct{}{
			"192.168.1.5": blank,
			"192.168.1.6": blank,
		}

		actual := make(map[string]struct{})
		for ipaddr := range common.GetAvailableIPAddresses(ip, ipNet, simAssignedIPAddresses) {
			actual[ipaddr] = blank
		}

		if !reflect.DeepEqual(assertResult, actual) {
			t.Fatalf("availableIP expected...%v, got ...%v\n", assertResult, actual)
		} else {
			t.Log("OK")
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
		options := Options{AccessToken: "", AccessTokenSecret: "", Zone: "is1a", CIDR: "192.168.1.0/29", MgwResourceID: "aaaaaaa"}
		_, _, err := validateArgs(options)
		if err != nil {
			t.Log("OK")
		} else {
			t.Fatalf("error is expected")
		}
	})
}
