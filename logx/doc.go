// Package logx 基于 log/slog 提供结构化日志初始化能力。
//
// 包的核心入口是 Options 和 New：调用方构造 Options，再交给 New 创建 logger。
// 默认输出到 stdout；File.Path 非空时启用文件轮转。
package logx
