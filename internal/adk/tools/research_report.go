package tools

import (
	"fmt"

	"github.com/run-bigpig/jcp/internal/services"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

// GetResearchReportInput 研报查询输入参数
type GetResearchReportInput struct {
	Code     string `json:"code" jsonschema:"股票代码，A股如 sz000001 或 000001；美股如 AAPL"`
	PageSize int    `json:"pageSize,omitzero" jsonschema:"每页数量，默认10"`
	PageNo   int    `json:"pageNo,omitzero" jsonschema:"页码，默认1"`
}

// GetResearchReportOutput 研报查询输出
type GetResearchReportOutput struct {
	Data       string `json:"data" jsonschema:"研报/推荐数据"`
	TotalCount int    `json:"totalCount" jsonschema:"总数量"`
}

// createResearchReportTool 创建研报查询工具
func (r *Registry) createResearchReportTool() (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetResearchReportInput) (GetResearchReportOutput, error) {
		fmt.Printf("[Tool:get_research_report] 调用开始, code=%s, pageSize=%d, pageNo=%d\n",
			input.Code, input.PageSize, input.PageNo)

		if input.Code == "" {
			fmt.Println("[Tool:get_research_report] 错误: 未提供股票代码")
			return GetResearchReportOutput{Data: "请提供股票代码"}, nil
		}

		// 美股使用 Finnhub 分析师推荐
		if services.IsUSStock(input.Code) {
			return r.getUSRecommendation(input.Code)
		}

		// A股使用东方财富研报
		pageSize := input.PageSize
		if pageSize == 0 {
			pageSize = 10
		}
		pageNo := input.PageNo
		if pageNo == 0 {
			pageNo = 1
		}

		result, err := r.researchReportService.GetResearchReports(input.Code, pageSize, pageNo)
		if err != nil {
			fmt.Printf("[Tool:get_research_report] 错误: %v\n", err)
			return GetResearchReportOutput{}, err
		}

		text := r.researchReportService.FormatReportsToText(result.Data)
		fmt.Printf("[Tool:get_research_report] 调用完成, 返回%d条研报\n", len(result.Data))

		return GetResearchReportOutput{
			Data:       text,
			TotalCount: result.TotalCount,
		}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_research_report",
		Description: "获取个股研报/分析师推荐。A股返回券商评级和研报；美股返回分析师买入/卖出/持有推荐趋势",
	}, handler)
}

// getUSRecommendation 获取美股分析师推荐
func (r *Registry) getUSRecommendation(symbol string) (GetResearchReportOutput, error) {
	recs, err := r.marketService.GetUSRecommendations(symbol)
	if err != nil {
		fmt.Printf("[Tool:get_research_report] 获取美股推荐失败: %v\n", err)
		return GetResearchReportOutput{}, err
	}

	if len(recs) == 0 {
		return GetResearchReportOutput{Data: "暂无分析师推荐数据"}, nil
	}

	var result string
	limit := len(recs)
	if limit > 6 {
		limit = 6
	}
	for i, rec := range recs[:limit] {
		result += fmt.Sprintf("%d. 【%s】%s\n", i+1, rec.Symbol, rec.Period)
		result += fmt.Sprintf("   强烈买入:%d | 买入:%d | 持有:%d | 卖出:%d | 强烈卖出:%d\n\n",
			rec.StrongBuy, rec.Buy, rec.Hold, rec.Sell, rec.StrongSell)
	}

	fmt.Printf("[Tool:get_research_report] 调用完成, 返回%d条美股推荐\n", limit)
	return GetResearchReportOutput{
		Data:       result,
		TotalCount: len(recs),
	}, nil
}

// GetReportContentInput 研报内容查询输入参数
type GetReportContentInput struct {
	InfoCode string `json:"infoCode" jsonschema:"研报唯一标识码，从研报列表中获取（仅A股）"`
}

// GetReportContentOutput 研报内容查询输出
type GetReportContentOutput struct {
	Content string `json:"content" jsonschema:"研报正文内容"`
	PDFUrl  string `json:"pdfUrl" jsonschema:"PDF下载链接"`
}

// createReportContentTool 创建研报内容查询工具
func (r *Registry) createReportContentTool() (tool.Tool, error) {
	handler := func(ctx tool.Context, input GetReportContentInput) (GetReportContentOutput, error) {
		fmt.Printf("[Tool:get_report_content] 调用开始, infoCode=%s\n", input.InfoCode)

		if input.InfoCode == "" {
			fmt.Println("[Tool:get_report_content] 错误: 未提供 infoCode")
			return GetReportContentOutput{Content: "请提供研报的 infoCode"}, nil
		}

		result, err := r.researchReportService.GetReportContent(input.InfoCode)
		if err != nil {
			fmt.Printf("[Tool:get_report_content] 错误: %v\n", err)
			return GetReportContentOutput{}, err
		}

		fmt.Printf("[Tool:get_report_content] 调用完成, 内容长度=%d\n", len(result.Content))

		return GetReportContentOutput{
			Content: result.Content,
			PDFUrl:  result.PDFUrl,
		}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:        "get_report_content",
		Description: "获取A股研报正文内容，需要先通过 get_research_report 获取研报列表中的 infoCode",
	}, handler)
}
