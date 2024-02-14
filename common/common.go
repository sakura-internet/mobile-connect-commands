package common

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
)

// セキュアモバイルコネクト SIM 詳細 API レスポンス
type SimAPIResponse struct {
	Sim []struct {
		ICCID string `json:"iccid"`
		IP    string `json:"ip"`
	} `json:"sim"`
	IsOK  bool `json:"is_ok"`
	Total int  `json:"Total"`
	From  int  `json:"From"`
	Count int  `json:"Count"`
}

// GetUsedIPAddressesInMGW
// MGW 内での利用されている IP アドレス一覧を取得する
// 利用されている IPアドレス一覧を取得
func GetUsedIPAddressesInMGW(accessToken string, accessTokenSecret string, zone string, mgwID string) (map[string]struct{}, error) {
	// ベーシック認証
	auth := fmt.Sprintf("%s:%s", accessToken, accessTokenSecret)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	headers := http.Header{
		"Authorization": []string{"Basic " + encodedAuth},
	}

	// モバイルゲートウェイ配下のSIMを全件取得するための URL の組み立て
	baseURL := fmt.Sprintf("https://secure.sakura.ad.jp/cloud/zone/%s/api/cloud/1.1/appliance/%s/mobilegateway/sims", zone, mgwID)
	queryParams := url.Values{}
	queryParams.Set("From", "0")
	fullURL := baseURL + "?" + queryParams.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("HTTPクライアントの初期化に失敗しました...%s", err.Error())
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTPクライアントの実行に失敗しました...%s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 認証情報の間違い
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("アクセストークン、アクセストークンシークレットを確認してください。SIM情報の取得に失敗しました")
		}

		// NotFound なので、モバイルゲートウェイのリソースID間違い
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("モバイルゲートウェイのリソースIDを確認してください。SIM情報の取得に失敗しました")
		}

		// その他のエラー
		return nil, fmt.Errorf("原因不明のエラーが発生しました...HTTPステータスコード: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("SIM情報のレスポンスの読み込みに失敗しました...%s", err.Error())
	}
	var apiResponse SimAPIResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("SIM情報のレスポンスのパースに失敗しました...%s", err.Error())
	}

	ipAddr := make(map[string]struct{})
	blank := struct{}{}
	for _, sim := range apiResponse.Sim {
		// map のキーを IPアドレスとし、キーのみ利用するので、バリューは空のstructとする
		ipAddr[sim.IP] = blank
	}

	return ipAddr, nil
}

// GetAvailableIPAddresses
// CIDR の パースした結果を受け取り、
// 利用されているIPアドレスと比較して、
// 利用可能な IP アドレス一覧を出力する
func GetAvailableIPAddresses(ip net.IP, ipNet *net.IPNet, usedIPAddresses map[string]struct{}) <-chan string {
	ipChan := make(chan string)
	broadcastAddr := broadcastAddress(ipNet)
	networkAddr := ip.Mask(ipNet.Mask).String()

	go func() {
		for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incrementIP(ip) {
			ipaddr := ip.String()
			// ネットワークアドレスはスキップする
			if ipaddr == networkAddr {
				continue
			}

			// ブロードキャストアドレスはスキップする
			if ipaddr == broadcastAddr {
				continue
			}

			// ipAddresses のキーに存在しなかったら追加する
			if _, exists := usedIPAddresses[ipaddr]; !exists {
				ipChan <- ipaddr
			}
		}
		close(ipChan)
	}()

	return ipChan
}

// ブロードキャストアドレスを返す
func broadcastAddress(ipNet *net.IPNet) string {
	// ipNet を参照渡しで渡してしまうと、元の値を書き換えてしまうので
	// コピーする
	copiedIP := make(net.IP, len(ipNet.IP))
	copy(copiedIP, ipNet.IP)

	for i := range copiedIP {
		/**
		192.168.1.0/24 の場合
		i == 0 のとき、ipNet.Mask == 255
		i == 1 のとき、ipNet.Mask == 255
		i == 2 のとき、ipNet.Mask == 255
		i == 3 のとき、ipNet.Mask == 0

		となり、ホスト部のビットを全部 1 にすれば良いので、ip[i] |= ^ipNet.Mask[i] で NOT 演算でビットを反転させる
		**/
		copiedIP[i] |= ^ipNet.Mask[i]
	}

	return copiedIP.String()
}

// IP アドレスを increment する
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
