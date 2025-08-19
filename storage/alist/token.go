package alist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	config "github.com/krau/SaveAny-Bot/config/storage"
)

func (a *Alist) getToken(ctx context.Context) error {
	loginBody, err := json.Marshal(a.loginInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, a.baseURL+"/api/auth/login", bytes.NewBuffer(loginBody))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	if loginResp.Code != http.StatusOK {
		return fmt.Errorf("%w: %s", ErrAlistLoginFailed, loginResp.Message)
	}

	// 线程安全地更新token和过期时间
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.updateTokenUnsafe(loginResp.Data.Token)
}

// updateTokenUnsafe 不安全地更新token（调用者必须持有锁）
func (a *Alist) updateTokenUnsafe(newToken string) error {
	oldToken := a.token
	a.token = newToken
	// 修正时间计算：配置中的TokenExp是秒数，不是小时数
	tokenExpSeconds := int64(a.config.TokenExp)
	if tokenExpSeconds <= 0 {
		tokenExpSeconds = 3600 // 默认1小时（3600秒）
	}
	a.tokenExpiry = time.Now().Add(time.Duration(tokenExpSeconds) * time.Second)
	
	// 更新认证状态
	a.authState.isAuthenticated = true
	a.authState.consecutiveFailures = 0
	a.authState.cooldownUntil = time.Time{}
	
	// 添加详细的token信息日志
	tokenChanged := oldToken != newToken
	if len(a.token) > 20 {
		tokenPreview := a.token[:10] + "..." + a.token[len(a.token)-10:]
		a.logger.Infof("Token updated successfully%s, preview: %s, expires at: %s", 
			map[bool]string{true: " (changed)", false: " (refreshed)"}[tokenChanged],
			tokenPreview, a.tokenExpiry.Format("2006-01-02 15:04:05"))
	} else {
		a.logger.Infof("Token updated successfully%s, length: %d chars, expires at: %s", 
			map[bool]string{true: " (changed)", false: " (refreshed)"}[tokenChanged],
			len(a.token), a.tokenExpiry.Format("2006-01-02 15:04:05"))
	}
	
	return nil
}

// getTokenUnsafe 不安全地获取新token（调用者必须持有锁）
func (a *Alist) getTokenUnsafe(ctx context.Context) error {
	loginBody, err := json.Marshal(a.loginInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, a.baseURL+"/api/auth/login", bytes.NewBuffer(loginBody))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	if loginResp.Code != http.StatusOK {
		return fmt.Errorf("%w: %s", ErrAlistLoginFailed, loginResp.Message)
	}

	// 不获取锁，直接更新（调用者必须已经持有锁）
	return a.updateTokenUnsafe(loginResp.Data.Token)
}

func (a *Alist) refreshToken(cfg config.AlistStorageConfig) {
	tokenExp := cfg.TokenExp
	if tokenExp <= 0 {
		a.logger.Warn("Invalid token expiration time, using default value")
		tokenExp = 3600
	}
	
	a.logger.Infof("Starting token refresh routine with interval: %d seconds", tokenExp)
	ticker := time.NewTicker(time.Duration(tokenExp) * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		a.logger.Debug("Attempting scheduled token refresh")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := a.getToken(ctx); err != nil {
			a.logger.Errorf("Failed to refresh jwt token: %v", err)
			cancel()
			// 如果刷新失败，等待较短时间后重试
			time.Sleep(30 * time.Second)
			continue
		}
		a.logger.Info("Scheduled token refresh completed successfully")
		cancel()
	}
}
