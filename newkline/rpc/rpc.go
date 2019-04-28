package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// APIFunc 是下单方法集合
type APIFunc struct {
	URL string
}

// CancelOrderOne 撤销单个挂单
func (api APIFunc) CancelOrderOne(memberID int64, orderID string) (string, error) {
	if memberID <= 0 {
		return "", errors.New("memberID error")
	}
	if orderID == "" {
		return "", errors.New("orderId error")
	}
	var v map[string]interface{}
	v["memberId"] = memberID
	v["orderId"] = orderID
	res, err := api.post("/exchange/order/create", v, "self")
	if err != nil {
		return "", err
	}
	return res, nil
}

// GetOrder 获取订单数
func (api APIFunc) GetOrder(coin string, memberID int64) (string, error) {
	v := make(map[string]interface{})
	v["order"] = "ASC"
	v["page"] = 1
	v["size"] = 100
	v["sort"] = "CREATED_DATE"
	v["query"] = make(map[string]interface{})
	ms := make(map[string]interface{})
	ms["coin"] = coin
	ms["memberId"] = memberID
	ms["trading"] = true
	v["query"] = ms
	res, err := api.post("/exchange/order/findPage", v, "query")
	if err != nil {
		return "", err
	}
	return res, nil
}

//CancelOrderAll 挂单撤销
func (api APIFunc) CancelOrderAll(memberID int64, arr []string) (string, error) {
	size := len(arr)
	if size <= 0 || size > 500 {
		size = 50
	}
	v := make(map[string]interface{})
	v["size"] = size
	v["orderList"] = arr
	v["memberId"] = memberID
	res, err := api.post("/exchange/cancel/saveBatch", v, "self")
	if err != nil {
		return "", err
	}
	return res, nil
}

// GetDepth 获取币种深度
func (api APIFunc) GetDepth(coin string) (string, error) {
	if coin == "" {
		return "", errors.New("coin is error")
	}
	res, err := api.get("", coin, "exchange", "https://open.bituan.cc/v1/get_ticker?symbol="+coin)
	if err != nil {
		return "", err
	}
	return res, nil
}

// GetMarket 获取币种挂单
func (api APIFunc) GetMarket(coin string) (string, error) {
	if coin == "" {
		return "", errors.New("coin is error")
	}
	res, err := api.get("", coin, "exchange", "https://open.bituan.cc/v1/market_dept?type=0&symbol="+coin)
	if err != nil {
		return "", err
	}
	return res, nil
}

// CreatOrder 下单方法
func (api APIFunc) CreatOrder(memberID int64, price float64, side string, symbol string, volume float64) (string, error) {
	//return "memberID:" + strconv.FormatInt(memberID, 10) + "price:" + strconv.FormatFloat(price, 'f', -1, 64) + "side:" + side + "symbol:" + symbol + "volume:" + strconv.FormatFloat(volume, 'f', -1, 64), nil
	if price <= 0 || volume <= 0 || memberID <= 0 {
		return "", errors.New("price or volume or memberID can`t < 0")
	}
	if len(side) <= 0 || len(symbol) <= 0 {
		return "", errors.New("side or symbol not null")
	}
	v := make(map[string]interface{})
	v["memberId"] = memberID
	v["price"] = price
	v["side"] = side
	v["source"] = "API"
	v["symbol"] = symbol
	v["type"] = "LIMIT"
	v["volume"] = volume
	res, err := api.post("/exchange/order/create", v, "self")
	if err != nil {
		return "", err
	}
	return res, nil
}

// GetExchangeData 获取其他交易所深度或者市价
func (api APIFunc) GetExchangeData(method, url string, v ...map[string]interface{}) (string, error) {
	if url == "" || method == "" {
		return "", errors.New("data is error")
	}
	switch method {
	case "get":
		return api.get("", "", "exchange", url)
	case "post":
		return api.post("", v[0], "exchange", url)
	default:
		return "", errors.New("method is error")
	}
}

// StringToData 数据格式化
func (api APIFunc) StringToData(str, types, name string) (map[string]interface{}, error) {
	if str == "" {
		return nil, errors.New("data is error")
	}
	var v interface{}
	err := json.Unmarshal([]byte(str), &v)
	if err != nil {
		return nil, errors.New("str to json error")
	}
	fmt.Println(types, name)
	switch types {
	case "trick":
		data := make(map[string]interface{})
		switch name {
		case "huobi":
			if v.(map[string]interface{})["status"] != "ok" {
				return nil, errors.New("str status is error")
			}
			data["now"] = v.(map[string]interface{})["tick"].(map[string]interface{})["close"].(float64)
			return data, nil
		case "ok":
			if _, ok := v.(map[string]interface{})["ticker"].(map[string]interface{})["last"]; !ok {
				return nil, errors.New("str status is error")
			}
			data["now"] = v.(map[string]interface{})["last"]
			return data, nil
		default:
			return nil, errors.New("name is error")
		}
	case "depth":
		var b, s []interface{}
		switch name {
		case "huobi":
			if v.(map[string]interface{})["status"] != "ok" {
				return nil, errors.New("str status is error")
			}
			b = v.(map[string]interface{})["tick"].(map[string]interface{})["bids"].([]interface{})
			s = v.(map[string]interface{})["tick"].(map[string]interface{})["asks"].([]interface{})

		case "ok":
			if v.(map[string]interface{})["asks"] == "" || v.(map[string]interface{})["bids"] == "" {
				return nil, errors.New("str status is error")
			}
			b = v.(map[string]interface{})["bids"].([]interface{})
			k := len(v.(map[string]interface{})["asks"].([]interface{})) - 1
			ask := make([]interface{}, k+1)
			x := 0
			for n := k; n >= 0; n-- {
				ask[x] = v.(map[string]interface{})["asks"].([]interface{})[n].([]interface{})
				x++
			}
			s = ask
		default:
			return nil, errors.New("name is error")
		}
		data := make(map[string]interface{})
		i, j := 0, 0
		bids := make([]float64, len(b))
		asks := make([]float64, len(s))
		for _, v := range b {
			bids[i] = v.([]interface{})[0].(float64)
			i++
		}
		for _, v := range s {
			asks[j] = v.([]interface{})[0].(float64)
			j++
		}
		data["bids"] = bids
		data["asks"] = asks
		return data, nil
	default:
		return nil, errors.New("types is error")
	}
}
func (api APIFunc) post(method string, v map[string]interface{}, types string, urls ...string) (string, error) {
	var url string
	if types == "exchange" {
		url = urls[0]
	} else if types == "self" {
		url = api.URL + method
	} else {
		url = "http://172.31.43.3:28007" + method
	}
	fmt.Println(url, v)
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Println("map to json error:", err)
		return "", err
	}
	res, err := http.NewRequest("POST", url, strings.NewReader(string(data)))
	if err != nil {
		fmt.Println("creat request error:", err)
		return "", err
	}
	defer res.Body.Close()
	res.Header.Add("content-type", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(res)
	if err != nil {
		fmt.Println("get body error:", err)
		return "", err
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read body error:", err)
		return "", err
	}
	return string(result), nil
}
func (api APIFunc) get(method, v, types string, urls ...string) (string, error) {
	var url string
	if types == "exchange" {
		url = urls[0]
	} else if types == "self" {
		url = api.URL + method + v
	} else {
		url = "http://172.31.43.3:28007" + method
	}
	fmt.Println(url)
	resp, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("creat request get error:", err)
		return "", err
	}
	//resp.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	//resp.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	//resp.Header.Add("Accept-Language", "zh-CN")
	//resp.Header.Add("Connection", "keep-alive")

	resp.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp.Header.Set("Accept-Language", "zh-cn")
	res, err := http.DefaultClient.Do(resp)
	if err != nil {
		fmt.Println("creat request get body error:", err)
		return "", err
	}
	defer res.Body.Close()
	result, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("creat request get  read body error:", err)
		return "", err
	}
	return string(result), nil
}
