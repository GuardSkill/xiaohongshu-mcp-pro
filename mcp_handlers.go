package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lisiyuan/xiaohongshu-mcp-pro/cookies"
	"github.com/lisiyuan/xiaohongshu-mcp-pro/xiaohongshu"
	"github.com/sirupsen/logrus"
)

// MCP 工具处理函数

// parseVisibility 从 MCP 参数中解析可见范围
func parseVisibility(args map[string]interface{}) string {
	v, ok := args["visibility"]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// handleCheckLoginStatus 处理检查登录状态
func (s *AppServer) handleCheckLoginStatus(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 检查登录状态")

	status, err := s.xiaohongshuService.CheckLoginStatus(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "检查登录状态失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 根据 IsLoggedIn 判断并返回友好的提示
	var resultText string
	if status.IsLoggedIn {
		resultText = fmt.Sprintf("✅ 已登录\n用户名: %s\n\n你可以使用其他功能了。", status.Username)
	} else {
		resultText = fmt.Sprintf("❌ 未登录\n\n请使用 get_login_qrcode 工具获取二维码进行登录。")
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleGetLoginQrcode 处理获取登录二维码请求。
// 返回二维码图片的 Base64 编码和超时时间，供前端展示扫码登录。
func (s *AppServer) handleGetLoginQrcode(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 获取登录扫码图片")

	result, err := s.xiaohongshuService.GetLoginQrcode(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取登录扫码图片失败: " + err.Error()}},
			IsError: true,
		}
	}

	if result.IsLoggedIn {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "你当前已处于登录状态"}},
		}
	}

	now := time.Now()
	deadline := func() string {
		d, err := time.ParseDuration(result.Timeout)
		if err != nil {
			return now.Format("2006-01-02 15:04:05")
		}
		return now.Add(d).Format("2006-01-02 15:04:05")
	}()

	// 已登录：文本 + 图片
	contents := []MCPContent{
		{Type: "text", Text: "请用小红书 App 在 " + deadline + " 前扫码登录 👇"},
		{
			Type:     "image",
			MimeType: "image/png",
			Data:     strings.TrimPrefix(result.Img, "data:image/png;base64,"),
		},
	}
	return &MCPToolResult{Content: contents}
}

// handleDeleteCookies 处理删除 cookies 请求，用于登录重置
func (s *AppServer) handleDeleteCookies(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 删除 cookies，重置登录状态")

	err := s.xiaohongshuService.DeleteCookies(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "删除 cookies 失败: " + err.Error()}},
			IsError: true,
		}
	}

	cookiePath := cookies.GetCookiesFilePath()
	resultText := fmt.Sprintf("Cookies 已成功删除，登录状态已重置。\n\n删除的文件路径: %s\n\n下次操作时，需要重新登录。", cookiePath)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handlePublishContent 处理发布内容
func (s *AppServer) handlePublishContent(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 发布内容")

	// 解析参数
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	imagePathsInterface, _ := args["images"].([]interface{})
	tagsInterface, _ := args["tags"].([]interface{})
	productsInterface, _ := args["products"].([]interface{})

	var imagePaths []string
	for _, path := range imagePathsInterface {
		if pathStr, ok := path.(string); ok {
			imagePaths = append(imagePaths, pathStr)
		}
	}

	var tags []string
	for _, tag := range tagsInterface {
		if tagStr, ok := tag.(string); ok {
			tags = append(tags, tagStr)
		}
	}

	var products []string
	for _, p := range productsInterface {
		if pStr, ok := p.(string); ok {
			products = append(products, pStr)
		}
	}

	// 解析定时发布参数
	scheduleAt, _ := args["schedule_at"].(string)
	visibility := parseVisibility(args)

	// 解析原创参数
	isOriginal, _ := args["is_original"].(bool)

	logrus.Infof("MCP: 发布内容 - 标题: %s, 图片数量: %d, 标签数量: %d, 定时: %s, 原创: %v, visibility: %s, 商品: %v", title, len(imagePaths), len(tags), scheduleAt, isOriginal, visibility, products)

	// 构建发布请求
	req := &PublishRequest{
		Title:      title,
		Content:    content,
		Images:     imagePaths,
		Tags:       tags,
		ScheduleAt: scheduleAt,
		IsOriginal: isOriginal,
		Visibility: visibility,
		Products:   products,
	}

	// 执行发布
	result, err := s.xiaohongshuService.PublishContent(ctx, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发布失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	resultText := fmt.Sprintf("内容发布成功: %+v", result)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handlePublishVideo 处理发布视频内容（仅本地单个视频文件）
func (s *AppServer) handlePublishVideo(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 发布视频内容（本地）")

	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	videoPath, _ := args["video"].(string)
	tagsInterface, _ := args["tags"].([]interface{})
	productsInterface, _ := args["products"].([]interface{})

	var tags []string
	for _, tag := range tagsInterface {
		if tagStr, ok := tag.(string); ok {
			tags = append(tags, tagStr)
		}
	}

	var products []string
	for _, p := range productsInterface {
		if pStr, ok := p.(string); ok {
			products = append(products, pStr)
		}
	}

	if videoPath == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发布失败: 缺少本地视频文件路径",
			}},
			IsError: true,
		}
	}

	// 解析定时发布参数
	scheduleAt, _ := args["schedule_at"].(string)
	visibility := parseVisibility(args)

	logrus.Infof("MCP: 发布视频 - 标题: %s, 标签数量: %d, 定时: %s, visibility: %s, 商品: %v", title, len(tags), scheduleAt, visibility, products)

	// 构建发布请求
	req := &PublishVideoRequest{
		Title:      title,
		Content:    content,
		Video:      videoPath,
		Tags:       tags,
		ScheduleAt: scheduleAt,
		Visibility: visibility,
		Products:   products,
	}

	// 执行发布
	result, err := s.xiaohongshuService.PublishVideo(ctx, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发布失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	resultText := fmt.Sprintf("视频发布成功: %+v", result)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleListFeeds 处理获取Feeds列表
func (s *AppServer) handleListFeeds(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 获取Feeds列表")

	result, err := s.xiaohongshuService.ListFeeds(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feeds列表失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取Feeds列表成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleSearchFeeds 处理搜索Feeds
func (s *AppServer) handleSearchFeeds(ctx context.Context, args SearchFeedsArgs) *MCPToolResult {
	logrus.Info("MCP: 搜索Feeds")

	if args.Keyword == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "搜索Feeds失败: 缺少关键词参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 搜索Feeds - 关键词: %s", args.Keyword)

	// 将 MCP 的 FilterOption 转换为 xiaohongshu.FilterOption
	filter := xiaohongshu.FilterOption{
		SortBy:      args.Filters.SortBy,
		NoteType:    args.Filters.NoteType,
		PublishTime: args.Filters.PublishTime,
		SearchScope: args.Filters.SearchScope,
		Location:    args.Filters.Location,
	}

	result, err := s.xiaohongshuService.SearchFeeds(ctx, args.Keyword, filter)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "搜索Feeds失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("搜索Feeds成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleGetFeedDetail 处理获取Feed详情
func (s *AppServer) handleGetFeedDetail(ctx context.Context, args map[string]any) *MCPToolResult {
	logrus.Info("MCP: 获取Feed详情")

	// 解析参数
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feed详情失败: 缺少feed_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feed详情失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	loadAll := false
	if raw, ok := args["load_all_comments"]; ok {
		switch v := raw.(type) {
		case bool:
			loadAll = v
		case string:
			if parsed, err := strconv.ParseBool(v); err == nil {
				loadAll = parsed
			}
		case float64:
			loadAll = v != 0
		}
	}

	// 解析评论配置参数，如果未提供则使用默认值
	config := xiaohongshu.DefaultCommentLoadConfig()

	if raw, ok := args["click_more_replies"]; ok {
		switch v := raw.(type) {
		case bool:
			config.ClickMoreReplies = v
		case string:
			if parsed, err := strconv.ParseBool(v); err == nil {
				config.ClickMoreReplies = parsed
			}
		}
	}

	if raw, ok := args["max_replies_threshold"]; ok {
		switch v := raw.(type) {
		case float64:
			config.MaxRepliesThreshold = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				config.MaxRepliesThreshold = parsed
			}
		case int:
			config.MaxRepliesThreshold = v
		}
	}

	if raw, ok := args["max_comment_items"]; ok {
		switch v := raw.(type) {
		case float64:
			config.MaxCommentItems = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				config.MaxCommentItems = parsed
			}
		case int:
			config.MaxCommentItems = v
		}
	}

	if raw, ok := args["scroll_speed"].(string); ok && raw != "" {
		config.ScrollSpeed = raw
	}

	logrus.Infof("MCP: 获取Feed详情 - Feed ID: %s, loadAllComments=%v, config=%+v", feedID, loadAll, config)

	result, err := s.xiaohongshuService.GetFeedDetailWithConfig(ctx, feedID, xsecToken, loadAll, config)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feed详情失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取Feed详情成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleUserProfile 获取用户主页
func (s *AppServer) handleUserProfile(ctx context.Context, args map[string]any) *MCPToolResult {
	logrus.Info("MCP: 获取用户主页")

	// 解析参数
	userID, ok := args["user_id"].(string)
	if !ok || userID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取用户主页失败: 缺少user_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取用户主页失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 获取用户主页 - User ID: %s", userID)

	result, err := s.xiaohongshuService.UserProfile(ctx, userID, xsecToken)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取用户主页失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取用户主页，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleLikeFeed 处理点赞/取消点赞
func (s *AppServer) handleLikeFeed(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少feed_id参数"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少xsec_token参数"}}, IsError: true}
	}
	unlike, _ := args["unlike"].(bool)

	var res *ActionResult
	var err error

	if unlike {
		res, err = s.xiaohongshuService.UnlikeFeed(ctx, feedID, xsecToken)
	} else {
		res, err = s.xiaohongshuService.LikeFeed(ctx, feedID, xsecToken)
	}

	if err != nil {
		action := "点赞"
		if unlike {
			action = "取消点赞"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + "失败: " + err.Error()}}, IsError: true}
	}

	action := "点赞"
	if unlike {
		action = "取消点赞"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s成功 - Feed ID: %s", action, res.FeedID)}}}
}

// handleFavoriteFeed 处理收藏/取消收藏
func (s *AppServer) handleFavoriteFeed(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少feed_id参数"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少xsec_token参数"}}, IsError: true}
	}
	unfavorite, _ := args["unfavorite"].(bool)

	var res *ActionResult
	var err error

	if unfavorite {
		res, err = s.xiaohongshuService.UnfavoriteFeed(ctx, feedID, xsecToken)
	} else {
		res, err = s.xiaohongshuService.FavoriteFeed(ctx, feedID, xsecToken)
	}

	if err != nil {
		action := "收藏"
		if unfavorite {
			action = "取消收藏"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + "失败: " + err.Error()}}, IsError: true}
	}

	action := "收藏"
	if unfavorite {
		action = "取消收藏"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s成功 - Feed ID: %s", action, res.FeedID)}}}
}

// handlePostComment 处理发表评论到Feed
func (s *AppServer) handlePostComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 发表评论到Feed")

	// 解析参数
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: 缺少feed_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: 缺少content参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 发表评论 - Feed ID: %s, 内容长度: %d", feedID, len(content))

	// 发表评论
	result, err := s.xiaohongshuService.PostCommentToFeed(ctx, feedID, xsecToken, content)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 返回成功结果，只包含feed_id
	resultText := fmt.Sprintf("评论发表成功 - Feed ID: %s", result.FeedID)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleReplyComment 处理回复评论
func (s *AppServer) handleReplyComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 回复评论")

	// 解析参数
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少feed_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	commentID, _ := args["comment_id"].(string)
	userID, _ := args["user_id"].(string)
	if commentID == "" && userID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少comment_id或user_id参数",
			}},
			IsError: true,
		}
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少content参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 回复评论 - Feed ID: %s, Comment ID: %s, User ID: %s, 内容长度: %d", feedID, commentID, userID, len(content))

	// 回复评论
	result, err := s.xiaohongshuService.ReplyCommentToFeed(ctx, feedID, xsecToken, commentID, userID, content)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 返回成功结果
	responseText := fmt.Sprintf("评论回复成功 - Feed ID: %s, Comment ID: %s, User ID: %s", result.FeedID, result.TargetCommentID, result.TargetUserID)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: responseText,
		}},
	}
}

// ===== 账号管理 handlers =====

// handleListAccounts 列出所有账号
func (s *AppServer) handleListAccounts(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 列出所有账号")

	resp := s.xiaohongshuService.ListAccounts()

	var lines []string
	lines = append(lines, fmt.Sprintf("当前激活账号: %s\n", resp.ActiveAccount))
	lines = append(lines, fmt.Sprintf("账号列表（共 %d 个）:", len(resp.Accounts)))
	for _, acc := range resp.Accounts {
		activeTag := ""
		if acc.IsActive {
			activeTag = " ✅ (激活)"
		}
		lines = append(lines, fmt.Sprintf("  - %s%s", acc.Name, activeTag))
		lines = append(lines, fmt.Sprintf("      cookies: %s", acc.CookiesFile))
		lines = append(lines, fmt.Sprintf("      最近使用: %s", acc.LastUsed.Format("2006-01-02 15:04:05")))
	}

	data, _ := json.Marshal(resp)
	return &MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: strings.Join(lines, "\n")},
			{Type: "text", Text: string(data)},
		},
	}
}

// handleSwitchAccount 切换激活账号
func (s *AppServer) handleSwitchAccount(ctx context.Context, name string) *MCPToolResult {
	logrus.Infof("MCP: 切换账号 -> %s", name)

	if name == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "切换账号失败: 账号名不能为空"}},
			IsError: true,
		}
	}

	if err := s.xiaohongshuService.SwitchAccount(name); err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "切换账号失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: fmt.Sprintf("已切换到账号 %q，后续操作将使用该账号的 cookies。", name),
		}},
	}
}

// handleAddAccount 添加新账号
func (s *AppServer) handleAddAccount(ctx context.Context, name string) *MCPToolResult {
	logrus.Infof("MCP: 添加账号 -> %s", name)

	if name == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "添加账号失败: 账号名不能为空"}},
			IsError: true,
		}
	}

	cookiesFile, err := s.xiaohongshuService.AddAccount(name)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "添加账号失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: fmt.Sprintf(
				"账号 %q 已添加，cookies 文件: %s\n已自动切换为激活账号，请立即使用 get_login_qrcode 扫码登录。",
				name, cookiesFile,
			),
		}},
	}
}

// handleRemoveAccount 删除账号
func (s *AppServer) handleRemoveAccount(ctx context.Context, name string) *MCPToolResult {
	logrus.Infof("MCP: 删除账号 -> %s", name)

	if name == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "删除账号失败: 账号名不能为空"}},
			IsError: true,
		}
	}

	if err := s.xiaohongshuService.RemoveAccount(name); err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "删除账号失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: fmt.Sprintf("账号 %q 已删除（包含其 cookies 文件）。", name),
		}},
	}
}

// handleBroadcastPublish 广播发布
func (s *AppServer) handleBroadcastPublish(ctx context.Context, args BroadcastPublishArgs) *MCPToolResult {
	logrus.Infof("MCP: 广播发布 -> 账号: %v, 标题: %s", args.Accounts, args.Title)

	req := &PublishRequest{
		Title:      args.Title,
		Content:    args.Content,
		Images:     args.Images,
		Tags:       args.Tags,
		ScheduleAt: args.ScheduleAt,
		IsOriginal: args.IsOriginal,
		Visibility: args.Visibility,
	}

	resp, err := s.xiaohongshuService.BroadcastPublish(ctx, args.Accounts, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "广播发布失败: " + err.Error()}},
			IsError: true,
		}
	}

	// 汇总文本
	var lines []string
	lines = append(lines, fmt.Sprintf("广播发布完成：成功 %d / 失败 %d", resp.SuccessCount, resp.FailCount))
	for _, r := range resp.Results {
		if r.Success {
			lines = append(lines, fmt.Sprintf("  ✅ %s: %s", r.AccountName, r.Message))
		} else {
			lines = append(lines, fmt.Sprintf("  ❌ %s: %s", r.AccountName, r.Error))
		}
	}

	data, _ := json.Marshal(resp)
	return &MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: strings.Join(lines, "\n")},
			{Type: "text", Text: string(data)},
		},
		IsError: resp.FailCount > 0 && resp.SuccessCount == 0,
	}
}

// handleCreatorPhoneLogin 发起 creator 手机号登录（发送验证码）
func (s *AppServer) handleCreatorPhoneLogin(ctx context.Context, phone string) *MCPToolResult {
	logrus.Infof("MCP: creator 手机号登录 -> phone: %s", phone)

	resp, err := s.xiaohongshuService.CreatorPhoneLogin(phone)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发送验证码失败: " + err.Error()}},
			IsError: true,
		}
	}

	// 返回截图 + 文字说明
	imgData := resp.Screenshot
	if idx := strings.Index(imgData, ","); idx >= 0 {
		imgData = imgData[idx+1:]
	}
	return &MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: resp.Message},
			{Type: "image", Data: imgData, MimeType: "image/png"},
		},
	}
}

// handleCreatorVerifyOTP 填写验证码完成 creator 登录
func (s *AppServer) handleCreatorVerifyOTP(ctx context.Context, otp string) *MCPToolResult {
	logrus.Infof("MCP: creator 验证码登录")

	if err := s.xiaohongshuService.CreatorVerifyOTP(otp); err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "验证码登录失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: "creator 登录成功，cookies 已保存。现在可以使用 publish_content 发布内容。"}},
	}
}
