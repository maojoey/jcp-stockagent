package services

import (
	"testing"
)

// TestGetStockRealTimeData 测试获取实时股票数据
func TestGetStockRealTimeData(t *testing.T) {
	ms := NewMarketService(nil)

	// 测试上海股票 (贵州茅台)
	t.Run("上海股票", func(t *testing.T) {
		stocks, err := ms.GetStockRealTimeData("sh600519")
		if err != nil {
			t.Fatalf("获取上海股票数据失败: %v", err)
		}
		if len(stocks) == 0 {
			t.Fatal("未获取到股票数据")
		}

		stock := stocks[0]
		t.Logf("股票: %s (%s)", stock.Name, stock.Symbol)
		t.Logf("价格: %.2f, 涨跌: %.2f (%.2f%%)", stock.Price, stock.Change, stock.ChangePercent)
		t.Logf("开盘: %.2f, 最高: %.2f, 最低: %.2f, 昨收: %.2f", stock.Open, stock.High, stock.Low, stock.PreClose)
		t.Logf("成交量: %d", stock.Volume)

		// 验证数据有效性
		if stock.Name == "" {
			t.Error("股票名称为空")
		}
		if stock.Price <= 0 {
			t.Logf("警告: 股票价格为0，可能是非交易时间")
		}
	})

	// 测试深圳股票 (平安银行)
	t.Run("深圳股票", func(t *testing.T) {
		stocks, err := ms.GetStockRealTimeData("sz000001")
		if err != nil {
			t.Fatalf("获取深圳股票数据失败: %v", err)
		}
		if len(stocks) == 0 {
			t.Fatal("未获取到股票数据")
		}

		stock := stocks[0]
		t.Logf("股票: %s (%s)", stock.Name, stock.Symbol)
		t.Logf("价格: %.2f", stock.Price)

		if stock.Name == "" {
			t.Error("股票名称为空")
		}
	})

	// 测试多只股票
	t.Run("多只股票", func(t *testing.T) {
		stocks, err := ms.GetStockRealTimeData("sh600519", "sz000001", "sh601318")
		if err != nil {
			t.Fatalf("获取多只股票数据失败: %v", err)
		}
		if len(stocks) != 3 {
			t.Errorf("期望获取3只股票，实际获取 %d 只", len(stocks))
		}

		for _, stock := range stocks {
			t.Logf("  %s: %.2f", stock.Name, stock.Price)
		}
	})
}

// TestGetStockDataWithOrderBook 测试获取股票数据含盘口
func TestGetStockDataWithOrderBook(t *testing.T) {
	ms := NewMarketService(nil)

	t.Run("获取盘口数据", func(t *testing.T) {
		data, err := ms.GetStockDataWithOrderBook("sh600519")
		if err != nil {
			t.Fatalf("获取盘口数据失败: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("未获取到数据")
		}

		stock := data[0]
		t.Logf("股票: %s, 价格: %.2f", stock.Name, stock.Price)

		// 检查买盘
		t.Logf("买盘 (%d 档):", len(stock.OrderBook.Bids))
		for i, bid := range stock.OrderBook.Bids {
			t.Logf("  买%d: 价格=%.2f, 量=%d手", i+1, bid.Price, bid.Size)
		}

		// 检查卖盘
		t.Logf("卖盘 (%d 档):", len(stock.OrderBook.Asks))
		for i, ask := range stock.OrderBook.Asks {
			t.Logf("  卖%d: 价格=%.2f, 量=%d手", i+1, ask.Price, ask.Size)
		}

		// 验证盘口数据
		if len(stock.OrderBook.Bids) == 0 {
			t.Log("警告: 买盘数据为空，可能是非交易时间")
		}
		if len(stock.OrderBook.Asks) == 0 {
			t.Log("警告: 卖盘数据为空，可能是非交易时间")
		}
	})
}

// TestGetKLineData 测试获取K线数据
func TestGetKLineData(t *testing.T) {
	ms := NewMarketService(nil)

	t.Run("日K线", func(t *testing.T) {
		data, err := ms.GetKLineData("sh600519", "1d", 10)
		if err != nil {
			t.Fatalf("获取K线数据失败: %v", err)
		}

		t.Logf("获取到 %d 条K线数据", len(data))
		if len(data) == 0 {
			t.Fatal("未获取到K线数据")
		}

		// 显示最近几条
		for i, k := range data {
			if i >= 3 {
				break
			}
			t.Logf("  %s: 开=%.2f 高=%.2f 低=%.2f 收=%.2f 量=%d",
				k.Time, k.Open, k.High, k.Low, k.Close, k.Volume)
		}

		// 验证数据
		k := data[0]
		if k.Time == "" {
			t.Error("K线时间为空")
		}
		if k.High < k.Low {
			t.Error("K线最高价小于最低价")
		}
	})
}
