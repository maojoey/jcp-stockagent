package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/run-bigpig/jcp/internal/models"
)

const (
	finnhubBaseURL = "https://finnhub.io/api/v1"
)

// finnhubQuoteResponse Finnhub 实时报价响应
type finnhubQuoteResponse struct {
	Current       float64 `json:"c"`  // 当前价格
	Change        float64 `json:"d"`  // 涨跌额
	ChangePercent float64 `json:"dp"` // 涨跌幅(%)
	High          float64 `json:"h"`  // 最高价
	Low           float64 `json:"l"`  // 最低价
	Open          float64 `json:"o"`  // 开盘价
	PrevClose     float64 `json:"pc"` // 昨收价
	Timestamp     int64   `json:"t"`  // 时间戳
}

// finnhubCandleResponse Finnhub K线响应
type finnhubCandleResponse struct {
	Close     []float64 `json:"c"`
	High      []float64 `json:"h"`
	Low       []float64 `json:"l"`
	Open      []float64 `json:"o"`
	Timestamp []int64   `json:"t"`
	Volume    []int64   `json:"v"`
	Status    string    `json:"s"` // "ok" or "no_data"
}

// finnhubSearchResponse Finnhub 股票搜索响应
type finnhubSearchResponse struct {
	Count  int `json:"count"`
	Result []struct {
		Description   string `json:"description"`
		DisplaySymbol string `json:"displaySymbol"`
		Symbol        string `json:"symbol"`
		Type          string `json:"type"`
	} `json:"result"`
}

// FinnhubRecommendation Finnhub 分析师推荐
type FinnhubRecommendation struct {
	Buy        int    `json:"buy"`
	Hold       int    `json:"hold"`
	Period     string `json:"period"`
	Sell       int    `json:"sell"`
	StrongBuy  int    `json:"strongBuy"`
	StrongSell int    `json:"strongSell"`
	Symbol     string `json:"symbol"`
}

// FinnhubCompanyNews Finnhub 公司新闻
type FinnhubCompanyNews struct {
	Category string `json:"category"`
	Datetime int64  `json:"datetime"`
	Headline string `json:"headline"`
	ID       int64  `json:"id"`
	Image    string `json:"image"`
	Related  string `json:"related"`
	Source   string `json:"source"`
	Summary  string `json:"summary"`
	URL      string `json:"url"`
}

// finnhubRequest 发起 Finnhub API 请求
func (ms *MarketService) finnhubRequest(endpoint string) ([]byte, error) {
	apiKey := ""
	if ms.configService != nil {
		apiKey = ms.configService.GetFinnhubAPIKey()
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Finnhub API Key 未配置，请在设置中配置")
	}

	url := fmt.Sprintf("%s%s&token=%s", finnhubBaseURL, endpoint, apiKey)
	resp, err := ms.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求 Finnhub 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Finnhub API 返回 %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	return body, nil
}

// getUSStockRealTimeData 获取美股实时数据
func (ms *MarketService) getUSStockRealTimeData(codes ...string) ([]models.Stock, error) {
	var stocks []models.Stock
	for _, code := range codes {
		body, err := ms.finnhubRequest(fmt.Sprintf("/quote?symbol=%s", code))
		if err != nil {
			log.Warn("获取美股 %s 实时数据失败: %v", code, err)
			continue
		}

		var quote finnhubQuoteResponse
		if err := json.Unmarshal(body, &quote); err != nil {
			log.Warn("解析美股 %s 报价失败: %v", code, err)
			continue
		}

		// 跳过无数据的情况
		if quote.Current == 0 && quote.PrevClose == 0 {
			continue
		}

		stocks = append(stocks, models.Stock{
			Symbol:        code,
			Name:          code, // Finnhub quote 不返回名称，使用代码
			Price:         quote.Current,
			Change:        quote.Change,
			ChangePercent: quote.ChangePercent,
			Open:          quote.Open,
			High:          quote.High,
			Low:           quote.Low,
			PreClose:      quote.PrevClose,
		})
	}
	return stocks, nil
}

// getUSKLineData 获取美股K线数据
func (ms *MarketService) getUSKLineData(code string, period string, days int) ([]models.KLineData, error) {
	resolution := usResolution(period)
	to := time.Now().Unix()
	from := to - int64(days)*24*60*60

	// 分时图(1m)获取当天数据
	if period == "1m" {
		resolution = "1"
		from = to - 24*60*60
	}

	body, err := ms.finnhubRequest(fmt.Sprintf("/stock/candle?symbol=%s&resolution=%s&from=%d&to=%d", code, resolution, from, to))
	if err != nil {
		return nil, err
	}

	var candle finnhubCandleResponse
	if err := json.Unmarshal(body, &candle); err != nil {
		return nil, fmt.Errorf("解析K线数据失败: %w", err)
	}

	if candle.Status == "no_data" || len(candle.Close) == 0 {
		return []models.KLineData{}, nil
	}

	klines := make([]models.KLineData, 0, len(candle.Close))
	for i := range candle.Close {
		t := time.Unix(candle.Timestamp[i], 0)
		var timeStr string
		if period == "1m" {
			timeStr = t.Format("2006-01-02 15:04:05")
		} else {
			timeStr = t.Format("2006-01-02")
		}

		klines = append(klines, models.KLineData{
			Time:   timeStr,
			Open:   candle.Open[i],
			High:   candle.High[i],
			Low:    candle.Low[i],
			Close:  candle.Close[i],
			Volume: candle.Volume[i],
		})
	}

	// 计算均线
	klines = calculateMA(klines, 5)
	klines = calculateMA(klines, 10)
	klines = calculateMA(klines, 20)

	return klines, nil
}

// getUSMarketIndices 获取美股三大指数
func (ms *MarketService) getUSMarketIndices() ([]models.MarketIndex, error) {
	// 美股三大指数 ETF 替代（Finnhub 免费版不支持直接查询指数）
	indexSymbols := []struct {
		Symbol string
		Name   string
	}{
		{"SPY", "标普500 (SPY)"},
		{"DIA", "道琼斯 (DIA)"},
		{"QQQ", "纳斯达克 (QQQ)"},
	}

	var indices []models.MarketIndex
	for _, idx := range indexSymbols {
		body, err := ms.finnhubRequest(fmt.Sprintf("/quote?symbol=%s", idx.Symbol))
		if err != nil {
			continue
		}

		var quote finnhubQuoteResponse
		if err := json.Unmarshal(body, &quote); err != nil {
			continue
		}
		if quote.Current == 0 {
			continue
		}

		indices = append(indices, models.MarketIndex{
			Code:          idx.Symbol,
			Name:          idx.Name,
			Price:         quote.Current,
			Change:        quote.Change,
			ChangePercent: quote.ChangePercent,
		})
	}
	return indices, nil
}

// GetUSMarketStatus 获取美股市场状态
func (ms *MarketService) GetUSMarketStatus() MarketStatus {
	// 美东时区 UTC-5 (EST) / UTC-4 (EDT)
	loc := time.FixedZone("EST", -5*60*60)
	now := time.Now().In(loc)

	// 周末休市
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return MarketStatus{
			Status:     "closed",
			StatusText: "周末休市",
			IsTradeDay: false,
		}
	}

	hour, minute := now.Hour(), now.Minute()
	currentMinutes := hour*60 + minute

	// 美股交易时间: 9:30-16:00 EST
	switch {
	case currentMinutes < 4*60:
		return MarketStatus{Status: "closed", StatusText: "休市", IsTradeDay: true}
	case currentMinutes < 9*60+30:
		return MarketStatus{Status: "pre_market", StatusText: "盘前交易", IsTradeDay: true}
	case currentMinutes < 16*60:
		return MarketStatus{Status: "trading", StatusText: "交易中", IsTradeDay: true}
	case currentMinutes < 20*60:
		return MarketStatus{Status: "pre_market", StatusText: "盘后交易", IsTradeDay: true}
	default:
		return MarketStatus{Status: "closed", StatusText: "已收盘", IsTradeDay: true}
	}
}

// SearchUSStocks 搜索美股
func (ms *MarketService) SearchUSStocks(keyword string) ([]StockSearchResult, error) {
	body, err := ms.finnhubRequest(fmt.Sprintf("/search?q=%s", keyword))
	if err != nil {
		return nil, err
	}

	var searchResp finnhubSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("解析搜索结果失败: %w", err)
	}

	var results []StockSearchResult
	for _, r := range searchResp.Result {
		// 只包含普通股
		if r.Type != "Common Stock" && r.Type != "ETP" {
			continue
		}
		results = append(results, StockSearchResult{
			Symbol:   r.Symbol,
			Name:     r.Description,
			Industry: r.Type,
			Market:   "US",
		})
	}
	return results, nil
}

// GetUSCompanyNews 获取美股个股新闻
func (ms *MarketService) GetUSCompanyNews(symbol string) ([]FinnhubCompanyNews, error) {
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -7).Format("2006-01-02")

	body, err := ms.finnhubRequest(fmt.Sprintf("/company-news?symbol=%s&from=%s&to=%s", symbol, from, to))
	if err != nil {
		return nil, err
	}

	var news []FinnhubCompanyNews
	if err := json.Unmarshal(body, &news); err != nil {
		return nil, fmt.Errorf("解析新闻数据失败: %w", err)
	}
	return news, nil
}

// GetUSGeneralNews 获取美股市场综合新闻
func (ms *MarketService) GetUSGeneralNews() ([]FinnhubCompanyNews, error) {
	body, err := ms.finnhubRequest("/news?category=general")
	if err != nil {
		return nil, err
	}

	var news []FinnhubCompanyNews
	if err := json.Unmarshal(body, &news); err != nil {
		return nil, fmt.Errorf("解析新闻数据失败: %w", err)
	}
	return news, nil
}

// GetUSRecommendations 获取美股分析师推荐
func (ms *MarketService) GetUSRecommendations(symbol string) ([]FinnhubRecommendation, error) {
	body, err := ms.finnhubRequest(fmt.Sprintf("/stock/recommendation?symbol=%s", symbol))
	if err != nil {
		return nil, err
	}

	var recs []FinnhubRecommendation
	if err := json.Unmarshal(body, &recs); err != nil {
		return nil, fmt.Errorf("解析推荐数据失败: %w", err)
	}
	return recs, nil
}

// usResolution 将周期转换为 Finnhub resolution 参数
func usResolution(period string) string {
	switch period {
	case "1m":
		return "1"
	case "1d":
		return "D"
	case "1w":
		return "W"
	case "1mo":
		return "M"
	default:
		return "D"
	}
}

// calculateMA 计算移动平均线
func calculateMA(klines []models.KLineData, n int) []models.KLineData {
	if len(klines) < n {
		return klines
	}
	for i := n - 1; i < len(klines); i++ {
		var sum float64
		for j := i - n + 1; j <= i; j++ {
			sum += klines[j].Close
		}
		ma := sum / float64(n)
		switch n {
		case 5:
			klines[i].MA5 = ma
		case 10:
			klines[i].MA10 = ma
		case 20:
			klines[i].MA20 = ma
		}
	}
	return klines
}
