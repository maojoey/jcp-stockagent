package tools

import (
	"fmt"
	"time"

	"github.com/run-bigpig/jcp/internal/services"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// GetNewsInput 快讯输入参数
type GetNewsInput struct {
	Limit  int    `json:"limit,omitzero" jsonschema:"返回条数，默认10条"`
	Symbol string `json:"symbol,omitempty" jsonschema:"美股代码（如 AAPL），提供时获取该股票相关新闻；不提供则获取A股财经快讯"`
}

// GetNewsOutput 快讯输出
type GetNewsOutput struct {
	Data string `json:"data" jsonschema:"财经快讯列表"`
}

// createNewsTool 创建快讯工具
func (r *Registry) createNewsTool() (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetNewsInput) (GetNewsOutput, error) {
		fmt.Printf("[Tool:get_news] 调用开始, limit=%d, symbol=%s\n", input.Limit, input.Symbol)

		limit := input.Limit
		if limit == 0 {
			limit = 10
		}

		// 如果提供了美股代码，获取美股新闻
		if input.Symbol != "" && services.IsUSStock(input.Symbol) {
			return r.getUSNews(input.Symbol, limit)
		}

		// 默认获取A股财联社快讯
		news, err := r.newsService.GetTelegraphList()
		if err != nil {
			fmt.Printf("[Tool:get_news] 错误: %v\n", err)
			return GetNewsOutput{}, err
		}

		if limit > len(news) {
			limit = len(news)
		}

		var result string
		for i := 0; i < limit; i++ {
			n := news[i]
			result += fmt.Sprintf("[%s] %s\n", n.Time, n.Content)
		}

		fmt.Printf("[Tool:get_news] 调用完成, 返回%d条快讯\n", limit)
		return GetNewsOutput{Data: result}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_news",
		Description: "获取财经新闻快讯。A股返回财联社快讯；提供美股代码(symbol)时返回该股票相关英文新闻",
	}, handler)
}

// getUSNews 获取美股新闻
func (r *Registry) getUSNews(symbol string, limit int) (GetNewsOutput, error) {
	news, err := r.marketService.GetUSCompanyNews(symbol)
	if err != nil {
		fmt.Printf("[Tool:get_news] 获取美股新闻失败: %v\n", err)
		return GetNewsOutput{}, err
	}

	if limit > len(news) {
		limit = len(news)
	}

	var result string
	for i := 0; i < limit; i++ {
		n := news[i]
		t := time.Unix(n.Datetime, 0).Format("01-02 15:04")
		result += fmt.Sprintf("[%s] [%s] %s\n", t, n.Source, n.Headline)
	}

	if result == "" {
		result = "暂无相关新闻"
	}

	fmt.Printf("[Tool:get_news] 调用完成, 返回%d条美股新闻\n", limit)
	return GetNewsOutput{Data: result}, nil
}
