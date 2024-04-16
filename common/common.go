package common

import (
	"bytes"
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

// BASIC認証のAuthorizationヘッダが設定されたhttp.Header{}インスタンスを作成
func createHeadersWithBasicAuth(user string, password string) http.Header {
	auth := fmt.Sprintf("%s:%s", user, password)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	headers := http.Header{
		"Authorization": []string{"Basic " + encodedAuth},
	}

	return headers
}

// GetUsedIPAddressesInMGW
// MGW 内での利用されている IP アドレス一覧を取得する
// 利用されている IPアドレス一覧を取得
func GetUsedIPAddressesInMGW(accessToken string, accessTokenSecret string, zone string, mgwID string) (map[string]struct{}, error) {
	// BASIC認証
	headers := createHeadersWithBasicAuth(accessToken, accessTokenSecret)

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

type SimRegisterInfo struct {
	ICCID    string
	PassCode string
}

// SIM作成APIのレスポンス
type SimCreateAPIResponse struct {
	CommonServiceItem struct {
		ID string `json:"ID"`
	}
	Success bool `json:"Success"`
	IsOK    bool `json:"is_ok"`
}

// APIのis_okレスポンス
type SimApiIsOkResponse struct {
	IsOK bool `json:"is_ok"`
}

// APIのis_fatalレスポンス
type SimApiIsFatalResponse struct {
	IsFatal   bool   `json:"is_fatal"`
	Serial    string `json:"serial"`
	Status    string `json:"status"`
	ErrorCode string `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

// SIMの作成
func createSim(accessToken string, accessTokenSecret string, simID string, simPasscode string) (string, error) {
	// BASIC認証
	headers := createHeadersWithBasicAuth(accessToken, accessTokenSecret)

	// SIM作成リクエストの組み立て
	baseURL := "https://secure.sakura.ad.jp/cloud/zone/is1a/api/cloud/1.1/commonserviceitem"
	body := fmt.Sprintf(`
	{
		"CommonServiceItem": {
			"Name": "%s",
			"Status": {
				"ICCID": "%s"
			},
			"Remark": {
				"PassCode": "%s"
			},
			"Provider": {
				"Class": "sim"
			}
		}
	}`, simID, simID, simPasscode)
	bytesBody := []byte(body)
	bufBody := bytes.NewBuffer(bytesBody)

	// リクエスト送信
	req, err := http.NewRequest("POST", baseURL, bufBody)
	if err != nil {
		return "", fmt.Errorf("HTTPクライアントの初期化に失敗しました...%s", err.Error())
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTPクライアントの実行に失敗しました...%s", err.Error())
	}
	defer resp.Body.Close()

	// レスポンスの確認
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("SIM作成のレスポンスの読み込みに失敗しました...%s", err.Error())
	}
	if resp.StatusCode != http.StatusCreated {
		// 認証情報の間違い
		if resp.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("アクセストークン、アクセストークンシークレットを確認してください。SIM作成に失敗しました")
		}

		// 作成済み
		if resp.StatusCode == http.StatusConflict {
			// エラーにしないけどリソースIDは空
			return "", nil
		}

		// 認証エラーじゃない場合
		var apiFatalRes SimApiIsFatalResponse
		err = json.Unmarshal(respBody, &apiFatalRes)
		if err != nil {
			return "", fmt.Errorf("SIM作成のレスポンスのパースに失敗しました...%s", err.Error())
		}
		// レスポンスに含まれているエラーメッセージを返す
		return "", fmt.Errorf("%s: (%s)%s", apiFatalRes.Serial, apiFatalRes.Status, apiFatalRes.ErrorMsg)
	}

	var apiResponse SimCreateAPIResponse
	err = json.Unmarshal(respBody, &apiResponse)
	if err != nil {
		return "", fmt.Errorf("SIM作成のレスポンスのパースに失敗しました...%s", err.Error())
	}

	// SIMのリソースIDを返す
	return apiResponse.CommonServiceItem.ID, nil
}

// SIMのIPアドレス設定
func assignIPAddressToSim(accessToken string, accessTokenSecret string, simID string, ipAddress string) error {
	// BASIC認証
	headers := createHeadersWithBasicAuth(accessToken, accessTokenSecret)

	// 『SIMのIPアドレス指定』のリクエストの組み立て
	baseURL := fmt.Sprintf("https://secure.sakura.ad.jp/cloud/zone/is1a/api/cloud/1.1/commonserviceitem/%s/sim/ip", simID)
	body := fmt.Sprintf(`{
	"sim": {
		"ip": "%s"
	}}`, ipAddress)
	bytesBody := []byte(body)
	bufBody := bytes.NewBuffer(bytesBody)

	// リクエスト送信
	req, err := http.NewRequest("PUT", baseURL, bufBody)
	if err != nil {
		return fmt.Errorf("HTTPクライアントの初期化に失敗しました...%s", err.Error())
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTPクライアントの実行に失敗しました...%s", err.Error())
	}
	defer resp.Body.Close()

	// レスポンスの確認
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("SIMのIPアドレス設定のレスポンスの読み込みに失敗しました...%s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		// 認証情報の間違い
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("アクセストークン、アクセストークンシークレットを確認してください。SIMのIPアドレス設定に失敗しました")
		}
		// 認証エラーじゃない場合
		var apiFatalRes SimApiIsFatalResponse
		err = json.Unmarshal(respBody, &apiFatalRes)
		if err != nil {
			return fmt.Errorf("SIMのIPアドレス設定のレスポンスのパースに失敗しました...%s", err.Error())
		}
		// レスポンスに含まれているエラーメッセージを返す
		return fmt.Errorf("%s: (%s)%s", apiFatalRes.Serial, apiFatalRes.Status, apiFatalRes.ErrorMsg)
	}

	// 設定の成否を返す
	var apiOkRes SimApiIsOkResponse
	err = json.Unmarshal(respBody, &apiOkRes)
	if err != nil {
		return fmt.Errorf("SIMのIPアドレス設定のレスポンスのパースに失敗しました...%s", err.Error())
	}

	if !apiOkRes.IsOK {
		return fmt.Errorf("SIMのIPアドレス設定が失敗しました")
	}

	return nil
}

// モバイルゲートウェイにSIMを登録
func assignSimToMgw(accessToken string, accessTokenSecret string, zone string, mgwID string, simID string) error {
	// BASIC認証
	headers := createHeadersWithBasicAuth(accessToken, accessTokenSecret)

	// 『モバイルゲートウェイにSIMを登録』のリクエストの組み立て
	baseURL := fmt.Sprintf("https://secure.sakura.ad.jp/cloud/zone/%s/api/cloud/1.1/appliance/%s/mobilegateway/sims", zone, mgwID)
	body := fmt.Sprintf(`{
	"sim": {
		"resource_id": "%s"
	}}`, simID)
	bytesBody := []byte(body)
	bufBody := bytes.NewBuffer(bytesBody)

	// リクエスト送信
	req, err := http.NewRequest("POST", baseURL, bufBody)
	if err != nil {
		return fmt.Errorf("HTTPクライアントの初期化に失敗しました...%s", err.Error())
	}
	req.Header = headers

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTPクライアントの実行に失敗しました...%s", err.Error())
	}
	defer resp.Body.Close()

	// レスポンスの確認
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("モバイルゲートウェイにSIMを登録のレスポンスの読み込みに失敗しました...%s", err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		// 認証情報の間違い
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("認証に失敗しました...%s", err.Error())
		}
		// 認証エラーじゃない場合
		var apiFatalRes SimApiIsFatalResponse
		err = json.Unmarshal(respBody, &apiFatalRes)
		if err != nil {
			return fmt.Errorf("モバイルゲートウェイにSIMを登録のレスポンスのパースに失敗しました...%s", err.Error())
		}
		// レスポンスに含まれているエラーメッセージを返す
		return fmt.Errorf("%s: (%s)%s", apiFatalRes.Serial, apiFatalRes.Status, apiFatalRes.ErrorMsg)
	}

	// 設定の成否を返す
	var apiOkRes SimApiIsOkResponse
	err = json.Unmarshal(respBody, &apiOkRes)
	if err != nil {
		return fmt.Errorf("モバイルゲートウェイにSIMを登録のレスポンスのパースに失敗しました...%s", err.Error())
	}

	if !apiOkRes.IsOK {
		return fmt.Errorf("SIMのIPアドレス設定が失敗しました")
	}

	return nil
}

// 　リスト内のSIMを登録する
func RegisterSimFromList(accessToken string, accessTokenSecret string, zone string, mgwID string, simList []SimRegisterInfo, ipList []string) error {

	if len(simList) > len(ipList) {
		return fmt.Errorf("登録対象のSIM %d 枚に対して割り当て可能なIPアドレスが %d 個しかありません", len(simList), len(ipList))
	}

	ipListIndex := 0
	for _, sim := range simList {
		// SIMを作成
		fmt.Printf("SIM登録(ICCID: %s)", sim.ICCID)
		simResourceId, err := createSim(accessToken, accessTokenSecret, sim.ICCID, sim.PassCode)
		if err != nil {
			fmt.Printf("[FAILED]\n")
			return err
		}
		if simResourceId == "" {
			//登録済みだからスキップ
			fmt.Printf("[SKIP]\n")
			continue
		}
		fmt.Printf("[OK]")

		// MGWにSIMを登録
		fmt.Printf(", モバイルゲートウェイに追加")
		err = assignSimToMgw(accessToken, accessTokenSecret, zone, mgwID, simResourceId)
		if err != nil {
			fmt.Printf("[FAILED]\n")
			return err
		}
		fmt.Printf("[OK]")

		// SIMにIPアドレスを設定
		fmt.Printf(", IPアドレスを設定(%s)", ipList[ipListIndex])
		err = assignIPAddressToSim(accessToken, accessTokenSecret, simResourceId, ipList[ipListIndex])
		if err != nil {
			fmt.Printf("[FAILED]\n")
			return err
		}
		fmt.Printf("[OK]\n")

		ipListIndex++
	}
	return nil
}
