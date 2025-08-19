package alist

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	config "github.com/krau/SaveAny-Bot/config/storage"
	"github.com/krau/SaveAny-Bot/pkg/ai"
	storenum "github.com/krau/SaveAny-Bot/pkg/enums/storage"
)

type authState struct {
	isAuthenticated  bool
	lastAuthAttempt  time.Time
	consecutiveFailures int
	cooldownUntil    time.Time
}

type Alist struct {
	client    *http.Client
	token       string
	tokenExpiry time.Time    // token过期时间
	baseURL     string
	loginInfo   *loginRequest
	config      config.AlistStorageConfig
	logger      *log.Logger
	mu          sync.RWMutex // 保护token的并发访问
	authMu      sync.Mutex   // 保护认证流程的串行化
	authState   authState    // 认证状态跟踪
}

func (a *Alist) Init(ctx context.Context, cfg config.StorageConfig) error {
	alistConfig, ok := cfg.(*config.AlistStorageConfig)
	if !ok {
		return fmt.Errorf("failed to cast alist config")
	}
	if err := alistConfig.Validate(); err != nil {
		return err
	}
	a.config = *alistConfig
	a.baseURL = alistConfig.URL
	a.client = getHttpClient()
	a.logger = log.FromContext(ctx).WithPrefix(fmt.Sprintf("alist[%s]", alistConfig.Name))
	
	// 初始化认证状态
	a.authState = authState{
		isAuthenticated:     false,
		consecutiveFailures: 0,
	}

	if alistConfig.Token != "" {
		a.token = alistConfig.Token
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/me", nil)
		if err != nil {
			a.logger.Fatalf("Failed to create request: %v", err)
			return err
		}
		req.Header.Set("Authorization", a.getBearerTokenSafe())

		resp, err := a.client.Do(req)
		if err != nil {
			a.logger.Fatalf("Failed to send request: %v", err)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			a.logger.Fatalf("Failed to get alist user info: %s", resp.Status)
			return err
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			a.logger.Fatalf("Failed to read response body: %v", err)
			return err
		}
		var meResp meResponse
		if err := json.Unmarshal(body, &meResp); err != nil {
			a.logger.Fatalf("Failed to unmarshal me response: %v", err)
			return err
		}
		if meResp.Code != http.StatusOK {
			a.logger.Fatalf("Failed to get alist user info: %s", meResp.Message)
			return err
		}
		a.logger.Debugf("Logged in Alist as %s", meResp.Data.Username)
		return nil
	}
	a.loginInfo = &loginRequest{
		Username: alistConfig.Username,
		Password: alistConfig.Password,
	}

	if err := a.getToken(ctx); err != nil {
		a.logger.Fatalf("Failed to login to Alist: %v", err)
		return err
	}
	a.logger.Debug("Logged in to Alist")

	go a.refreshToken(*alistConfig)
	return nil
}

// isAuthError 检查错误是否为认证相关错误
func (a *Alist) isAuthError(code int) bool {
	// Alist的认证错误码，包括40140116等
	authErrorCodes := []int{40140116, 401, 403}
	for _, authCode := range authErrorCodes {
		if code == authCode {
			return true
		}
	}
	return false
}

// isAuthErrorInMessage 检查错误消息中是否包含认证错误
func (a *Alist) isAuthErrorInMessage(message string) bool {
	// 检查消息中是否包含认证相关的错误码或文本
	return strings.Contains(message, "40140116") || 
		   strings.Contains(message, "no auth") ||
		   strings.Contains(message, "unauthorized") ||
		   strings.Contains(message, "authentication failed") ||
		   strings.Contains(message, "token is invalidated") ||
		   strings.Contains(message, "token expired") ||
		   strings.Contains(message, "invalid token")
}

// ensureAuth 确保当前token有效，如果无效则重新认证
func (a *Alist) ensureAuth(ctx context.Context) error {
	// 使用专门的认证锁确保认证过程串行化
	a.authMu.Lock()
	defer a.authMu.Unlock()
	
	// 检查是否在冷却期
	if time.Now().Before(a.authState.cooldownUntil) {
		cooldownRemaining := time.Until(a.authState.cooldownUntil)
		a.logger.Warnf("Authentication in cooldown, %v remaining", cooldownRemaining)
		return fmt.Errorf("authentication cooldown active, %v remaining", cooldownRemaining)
	}
	
	// 如果没有登录信息，无法重新认证
	if a.loginInfo == nil {
		a.authState.consecutiveFailures++
		return fmt.Errorf("no login information available for re-authentication")
	}
	
	// 计算退避时间
	backoffDuration := a.calculateBackoff(a.authState.consecutiveFailures)
	if !a.authState.lastAuthAttempt.IsZero() {
		timeSinceLastAttempt := time.Since(a.authState.lastAuthAttempt)
		if timeSinceLastAttempt < backoffDuration {
			waitTime := backoffDuration - timeSinceLastAttempt
			a.logger.Infof("Applying authentication backoff, waiting %v", waitTime)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
				// 继续执行
			}
		}
	}
	
	a.logger.Infof("Attempting to re-authenticate with Alist (attempt %d)", a.authState.consecutiveFailures+1)
	a.authState.lastAuthAttempt = time.Now()
	
	// 获取写锁来更新token
	a.mu.Lock()
	err := a.getTokenUnsafe(ctx)
	a.mu.Unlock()
	
	if err != nil {
		a.authState.consecutiveFailures++
		a.authState.isAuthenticated = false
		
		// 如果连续失败次数过多，进入冷却期
		if a.authState.consecutiveFailures >= 5 {
			cooldownDuration := time.Duration(a.authState.consecutiveFailures-4) * 5 * time.Minute
			a.authState.cooldownUntil = time.Now().Add(cooldownDuration)
			a.logger.Errorf("Too many consecutive auth failures (%d), entering cooldown for %v", 
				a.authState.consecutiveFailures, cooldownDuration)
		}
		
		a.logger.Errorf("Failed to re-authenticate (failure %d): %v", a.authState.consecutiveFailures, err)
		return fmt.Errorf("failed to re-authenticate (attempt %d): %w", a.authState.consecutiveFailures, err)
	}
	
	// 认证成功，重置状态
	a.authState.consecutiveFailures = 0
	a.authState.isAuthenticated = true
	a.authState.cooldownUntil = time.Time{}
	
	a.logger.Info("Successfully re-authenticated with Alist")
	return nil
}

// calculateBackoff 计算退避时间
func (a *Alist) calculateBackoff(failureCount int) time.Duration {
	if failureCount == 0 {
		return 0
	}
	
	// 指数退避：1s, 2s, 4s, 8s, 16s，最大60s
	backoff := time.Duration(1<<uint(min(failureCount, 6))) * time.Second
	if backoff > 60*time.Second {
		backoff = 60 * time.Second
	}
	
	// 添加抖动（±25%）
	jitterRange := float64(backoff) * 0.25
	jitter := time.Duration(jitterRange * (2*rand.Float64() - 1))
	
	result := backoff + jitter
	if result < 0 {
		result = backoff // 确保不返回负值
	}
	
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getTokenSafe 线程安全地获取token
func (a *Alist) getTokenSafe() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.token
}

// getBearerTokenSafe 线程安全地获取标准格式的Bearer token
func (a *Alist) getBearerTokenSafe() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.token == "" {
		return ""
	}
	// 确保token以Bearer 开头，如果没有则添加
	if strings.HasPrefix(a.token, "Bearer ") {
		return a.token
	}
	return "Bearer " + a.token
}

// isTokenExpiringSoon 检查token是否即将过期（5分钟内）
func (a *Alist) isTokenExpiringSoon() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.tokenExpiry.IsZero() {
		return false // 如果没有设置过期时间，不进行主动刷新
	}
	return time.Until(a.tokenExpiry) < 5*time.Minute
}

// validateToken 验证token的有效性，通过调用me接口
func (a *Alist) validateToken(ctx context.Context) error {
	a.mu.RLock()
	token := a.token
	a.mu.RUnlock()
	
	if token == "" {
		return fmt.Errorf("no token available")
	}
	
	// 创建带超时的子上下文
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(validateCtx, http.MethodGet, a.baseURL+"/api/me", nil)
	if err != nil {
		return fmt.Errorf("failed to create token validation request: %w", err)
	}
	req.Header.Set("Authorization", a.getBearerTokenSafe())
	
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应体用于错误分析
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token validation response: %w", err)
	}
	
	// 解析响应
	var meResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &meResp); err != nil {
		return fmt.Errorf("failed to parse token validation response: %w", err)
	}
	
	if meResp.Code != http.StatusOK {
		if a.isAuthError(meResp.Code) || a.isAuthErrorInMessage(meResp.Message) {
			return fmt.Errorf("token validation failed - auth error (code: %d, message: %s)", meResp.Code, meResp.Message)
		}
		return fmt.Errorf("token validation failed (code: %d, message: %s)", meResp.Code, meResp.Message)
	}
	
	a.logger.Debug("Token validation successful")
	return nil
}

// ensureTokenValid 确保token有效，如果即将过期或无效则刷新
func (a *Alist) ensureTokenValid(ctx context.Context) error {
	// 检查是否即将过期
	if a.isTokenExpiringSoon() {
		a.logger.Info("Token expiring soon, refreshing proactively")
		return a.ensureAuth(ctx)
	}
	
	// 检查认证状态
	a.mu.RLock()
	isAuthenticated := a.authState.isAuthenticated
	a.mu.RUnlock()
	
	// 如果之前认证失败，验证token是否真的有效
	if !isAuthenticated {
		a.logger.Debug("Previous authentication failed, validating token")
		if err := a.validateToken(ctx); err != nil {
			a.logger.Infof("Token validation failed: %v, attempting re-authentication", err)
			return a.ensureAuth(ctx)
		}
		// 验证成功，更新状态
		a.mu.Lock()
		a.authState.isAuthenticated = true
		a.mu.Unlock()
	}
	
	return nil
}

func (a *Alist) Type() storenum.StorageType {
	return storenum.Alist
}

func (a *Alist) Name() string {
	return a.config.Name
}

func (a *Alist) Save(ctx context.Context, reader io.Reader, storagePath string) error {
	a.logger.Infof("Saving file to %s", storagePath)
	
	// 路径安全验证和清理
	if err := ai.ValidateStoragePath(storagePath); err != nil {
		a.logger.Warnf("Storage path validation failed: %v, attempting to sanitize", err)
		storagePath = ai.SanitizeStoragePath(storagePath)
		a.logger.Infof("Sanitized storage path: %s", storagePath)
		
		// 再次验证清理后的路径
		if err := ai.ValidateStoragePath(storagePath); err != nil {
			return fmt.Errorf("storage path still invalid after sanitization: %w", err)
		}
	}
	
	// 主动检查token有效性
	if err := a.ensureTokenValid(ctx); err != nil {
		a.logger.Warnf("Failed to ensure token validity: %v", err)
	}

	// 简化目录检查：只对深层嵌套路径进行检查，减少认证失败点
	dir := path.Dir(storagePath)
	// 只对超过两级深度的路径进行检查，减少不必要的API调用
	pathDepth := len(strings.Split(strings.Trim(dir, "/"), "/"))
	if dir != "." && dir != "/" && pathDepth > 2 {
		if !a.ensureDirectoryAccessible(ctx, dir) {
			a.logger.Warnf("Deep directory may not be accessible: %s, relying on Alist auto-creation", dir)
			// 不返回错误，完全依靠Alist的自动目录创建能力
		}
	}

	ext := path.Ext(storagePath)
	base := strings.TrimSuffix(storagePath, ext)
	candidate := storagePath
	for i := 1; a.existsWithRetry(ctx, candidate); i++ {
		candidate = fmt.Sprintf("%s_%d%s", base, i, ext)
	}

	return a.saveWithRetry(ctx, reader, candidate)
}

// ensureDirectoryAccessible 检查目录是否可访问（不抛出错误，仅记录警告）
func (a *Alist) ensureDirectoryAccessible(ctx context.Context, dirPath string) bool {
	maxRetries := 2
	var lastError error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			a.logger.Infof("Retrying directory accessibility check (attempt %d/%d) for %s", attempt+1, maxRetries+1, dirPath)
		}

		body := map[string]any{
			"path":     dirPath,
			"password": "",
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			lastError = fmt.Errorf("failed to marshal directory check request: %w", err)
			a.logger.Debugf("Marshal error on attempt %d: %v", attempt+1, err)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/fs/get", bytes.NewBuffer(bodyBytes))
		if err != nil {
			lastError = fmt.Errorf("failed to create directory check request: %w", err)
			a.logger.Debugf("Request creation error on attempt %d: %v", attempt+1, err)
			continue
		}
		req.Header.Set("Authorization", a.getBearerTokenSafe())
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			lastError = fmt.Errorf("failed to send directory check request: %w", err)
			a.logger.Debugf("HTTP request error on attempt %d: %v", attempt+1, err)
			if attempt == maxRetries {
				a.logger.Warnf("Directory check failed after %d attempts, last error: %v", maxRetries+1, lastError)
			}
			continue
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastError = fmt.Errorf("failed to read directory check response: %w", err)
			a.logger.Debugf("Response read error on attempt %d: %v", attempt+1, err)
			if attempt == maxRetries {
				a.logger.Warnf("Directory check failed after %d attempts, last error: %v", maxRetries+1, lastError)
			}
			continue
		}

		var fsGetResp fsGetResponse
		if err := json.Unmarshal(data, &fsGetResp); err != nil {
			lastError = fmt.Errorf("failed to unmarshal directory check response: %w", err)
			a.logger.Debugf("JSON unmarshal error on attempt %d: %v", attempt+1, err)
			if attempt == maxRetries {
				a.logger.Warnf("Directory check failed after %d attempts, last error: %v", maxRetries+1, lastError)
			}
			continue
		}

		if fsGetResp.Code == http.StatusOK {
			a.logger.Debugf("Directory check successful for %s on attempt %d", dirPath, attempt+1)
			return true
		}

		// 检查是否为认证错误
		if a.isAuthError(fsGetResp.Code) || a.isAuthErrorInMessage(fsGetResp.Message) {
			lastError = fmt.Errorf("authentication error during directory check (code: %d, message: %s)", fsGetResp.Code, fsGetResp.Message)
			a.logger.Warnf("Authentication error when checking directory on attempt %d (code: %d, message: %s), attempting re-authentication", 
				attempt+1, fsGetResp.Code, fsGetResp.Message)
			
			if err := a.ensureAuth(ctx); err != nil {
				a.logger.Errorf("Failed to re-authenticate for directory check on attempt %d: %v", attempt+1, err)
				lastError = fmt.Errorf("re-authentication failed: %w", err)
				if attempt == maxRetries {
					a.logger.Warnf("Directory check failed after %d attempts, last error: %v", maxRetries+1, lastError)
				}
				continue
			}
			a.logger.Debug("Re-authentication successful, retrying directory check")
			// 重新认证成功，继续重试
			continue
		}

		// 其他错误，记录并跳出
		lastError = fmt.Errorf("directory check failed (code: %d, message: %s)", fsGetResp.Code, fsGetResp.Message)
		a.logger.Debugf("Directory check non-auth error on attempt %d (code: %d): %s", attempt+1, fsGetResp.Code, fsGetResp.Message)
		if attempt == maxRetries {
			a.logger.Warnf("Directory check failed after %d attempts, last error: %v", maxRetries+1, lastError)
		}
		break // 对于非认证错误，不再重试
	}
	
	a.logger.Debugf("Directory check ultimately failed for %s: %v", dirPath, lastError)
	return false
}

// saveWithRetry 带重试的文件保存，支持自动重新认证
func (a *Alist) saveWithRetry(ctx context.Context, reader io.Reader, candidate string) error {
	// 需要先读取reader到内存，因为可能需要重试
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			a.logger.Infof("Retrying file upload (attempt %d/%d)", attempt+1, maxRetries+1)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.baseURL+"/api/fs/put", bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", a.getBearerTokenSafe())
		req.Header.Set("File-Path", url.PathEscape(candidate))
		req.Header.Set("Content-Type", "application/octet-stream")
		req.ContentLength = int64(len(data))

		resp, err := a.client.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to send request: %w", err)
			}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to read response body: %w", err)
			}
			continue
		}

		var putResp putResponse
		if err := json.Unmarshal(body, &putResp); err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to unmarshal put response: %w", err)
			}
			continue
		}

		if putResp.Code == http.StatusOK {
			a.logger.Infof("File saved successfully to %s", candidate)
			return nil
		}

		// 检查是否为认证错误（代码或消息中）
		if a.isAuthError(putResp.Code) || a.isAuthErrorInMessage(putResp.Message) {
			a.logger.Warnf("Authentication error detected (code: %d, message: %s), attempting re-authentication", putResp.Code, putResp.Message)
			if err := a.ensureAuth(ctx); err != nil {
				if attempt == maxRetries {
					return fmt.Errorf("failed to re-authenticate: %w", err)
				}
				continue
			}
			// 重新认证成功，继续重试
			continue
		}

		// 其他错误
		if attempt == maxRetries {
			return fmt.Errorf("failed to save file to Alist: %d, %s", putResp.Code, putResp.Message)
		}
	}

	return fmt.Errorf("failed to save file after %d attempts", maxRetries+1)
}

func (a *Alist) JoinStoragePath(p string) string {
	return path.Join(a.config.BasePath, p)
}

func (a *Alist) Exists(ctx context.Context, storagePath string) bool {
	return a.existsWithRetry(ctx, storagePath)
}

// existsWithRetry 带重试的文件存在性检查，支持自动重新认证
// 使用正确的API: 先获取文件信息，如果404则检查是否为目录
func (a *Alist) existsWithRetry(ctx context.Context, storagePath string) bool {
	maxRetries := 2
	var lastError error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			a.logger.Debugf("Retrying file existence check (attempt %d/%d) for %s", attempt+1, maxRetries+1, storagePath)
		}

		// 直接获取文件/目录信息
		body := map[string]any{
			"path":     storagePath,
			"password": "",
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			lastError = fmt.Errorf("failed to marshal request body: %w", err)
			a.logger.Debugf("Marshal error on existence check attempt %d: %v", attempt+1, err)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/fs/get", bytes.NewBuffer(bodyBytes))
		if err != nil {
			lastError = fmt.Errorf("failed to create request: %w", err)
			a.logger.Debugf("Request creation error on existence check attempt %d: %v", attempt+1, err)
			continue
		}
		req.Header.Set("Authorization", a.getBearerTokenSafe())
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			lastError = fmt.Errorf("failed to send request: %w", err)
			a.logger.Debugf("HTTP request error on existence check attempt %d: %v", attempt+1, err)
			if attempt == maxRetries {
				a.logger.Warnf("File existence check failed after %d attempts for %s, last error: %v", maxRetries+1, storagePath, lastError)
			}
			continue
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastError = fmt.Errorf("failed to read response body: %w", err)
			a.logger.Debugf("Response read error on existence check attempt %d: %v", attempt+1, err)
			if attempt == maxRetries {
				a.logger.Warnf("File existence check failed after %d attempts for %s, last error: %v", maxRetries+1, storagePath, lastError)
			}
			continue
		}

		var fsGetResp fsGetResponse
		if err := json.Unmarshal(data, &fsGetResp); err != nil {
			lastError = fmt.Errorf("failed to unmarshal fs get response: %w", err)
			a.logger.Debugf("JSON unmarshal error on existence check attempt %d: %v", attempt+1, err)
			if attempt == maxRetries {
				a.logger.Warnf("File existence check failed after %d attempts for %s, last error: %v", maxRetries+1, storagePath, lastError)
			}
			continue
		}

		if fsGetResp.Code == http.StatusOK {
			a.logger.Debugf("File existence confirmed for %s on attempt %d", storagePath, attempt+1)
			return true
		}

		// 检查是否为认证错误（代码或消息中）
		if a.isAuthError(fsGetResp.Code) || a.isAuthErrorInMessage(fsGetResp.Message) {
			lastError = fmt.Errorf("authentication error during existence check (code: %d, message: %s)", fsGetResp.Code, fsGetResp.Message)
			a.logger.Warnf("Authentication error detected in existence check on attempt %d (code: %d, message: %s), attempting re-authentication", 
				attempt+1, fsGetResp.Code, fsGetResp.Message)
			
			if err := a.ensureAuth(ctx); err != nil {
				a.logger.Errorf("Failed to re-authenticate for existence check on attempt %d: %v", attempt+1, err)
				lastError = fmt.Errorf("re-authentication failed: %w", err)
				if attempt == maxRetries {
					a.logger.Warnf("File existence check failed after %d attempts for %s, last error: %v", maxRetries+1, storagePath, lastError)
				}
				continue
			}
			a.logger.Debug("Re-authentication successful, retrying existence check")
			// 重新认证成功，继续重试
			continue
		}

		// 404说明文件不存在，这是正常情况
		if fsGetResp.Code == 404 {
			a.logger.Debugf("File does not exist (404): %s", storagePath)
			return false
		}

		// 其他错误，记录详细信息
		lastError = fmt.Errorf("existence check failed (code: %d, message: %s)", fsGetResp.Code, fsGetResp.Message)
		a.logger.Debugf("File existence check non-auth error on attempt %d (code: %d): %s", attempt+1, fsGetResp.Code, fsGetResp.Message)
		if attempt == maxRetries {
			a.logger.Warnf("File existence check failed after %d attempts for %s, last error: %v", maxRetries+1, storagePath, lastError)
		}
		break // 对于非认证错误，不再重试
	}

	a.logger.Debugf("File existence check ultimately failed for %s: %v", storagePath, lastError)
	return false
}

// Impl StorageCannotStream interface
func (a *Alist) CannotStream() string {
	return "Alist does not support chunked transfer encoding"
}
