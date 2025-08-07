package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/eolinker/ai-prompt-proxy/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务
type AuthService struct {
	dbManager  *db.Manager
	jwtSecret  []byte
	rsaPrivKey *rsa.PrivateKey
}

// Claims JWT声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// EncryptedLoginRequest 加密的登录请求
type EncryptedLoginRequest struct {
	Username          string `json:"username" binding:"required"`
	EncryptedPassword string `json:"encrypted_password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// EncryptedRegisterRequest 加密的注册请求
type EncryptedRegisterRequest struct {
	Username          string `json:"username" binding:"required"`
	EncryptedPassword string `json:"encrypted_password" binding:"required"`
}

// PublicKeyResponse 公钥响应
type PublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string   `json:"token"`
	User      *db.User `json:"user"`
	ExpiresAt int64    `json:"expires_at"`
}

// NewAuthService 创建认证服务
func NewAuthService(dbManager *db.Manager) (*AuthService, error) {
	// 生成或获取JWT密钥
	secret, err := getOrCreateJWTSecret(dbManager)
	if err != nil {
		return nil, fmt.Errorf("获取JWT密钥失败: %w", err)
	}

	// 生成RSA密钥对
	rsaPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("生成RSA密钥失败: %w", err)
	}

	return &AuthService{
		dbManager:  dbManager,
		jwtSecret:  secret,
		rsaPrivKey: rsaPrivKey,
	}, nil
}

// getOrCreateJWTSecret 获取或创建JWT密钥
func getOrCreateJWTSecret(dbManager *db.Manager) ([]byte, error) {
	// 尝试从数据库获取现有密钥
	secretStr, err := dbManager.GetMetadata("jwt_secret")
	if err == nil {
		// 密钥存在，解码并返回
		return base64.StdEncoding.DecodeString(secretStr)
	}

	// 密钥不存在，生成新的密钥
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("生成JWT密钥失败: %w", err)
	}

	// 将密钥保存到数据库
	secretStr = base64.StdEncoding.EncodeToString(secret)
	if err := dbManager.SetMetadata("jwt_secret", secretStr); err != nil {
		return nil, fmt.Errorf("保存JWT密钥失败: %w", err)
	}

	return secret, nil
}

// HashPassword 加密密码
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func (s *AuthService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GenerateToken 生成JWT token
func (s *AuthService) GenerateToken(user *db.User) (string, int64, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // 24小时过期
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, fmt.Errorf("生成token失败: %w", err)
	}

	return tokenString, expirationTime.Unix(), nil
}

// ValidateToken 验证JWT token
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("解析token失败: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token无效")
	}

	return claims, nil
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// 获取用户
	user, err := s.dbManager.GetUserByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 检查用户是否被禁用
	if !user.IsEnabled {
		return nil, fmt.Errorf("用户已被禁用")
	}

	// 验证密码
	if !s.CheckPassword(user.Password, req.Password) {
		return nil, fmt.Errorf("用户名或密码错误")
	}

	// 更新最后登录时间
	if err := s.dbManager.UpdateUserLastLogin(user.ID); err != nil {
		// 记录错误但不影响登录流程
		fmt.Printf("更新用户最后登录时间失败: %v\n", err)
	}

	// 生成token
	token, expiresAt, err := s.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成token失败: %w", err)
	}

	return &LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}, nil
}

// Register 用户注册（仅在首次安装时允许）
func (s *AuthService) Register(req *RegisterRequest) (*LoginResponse, error) {
	// 检查是否已有用户
	count, err := s.dbManager.GetUserCount()
	if err != nil {
		return nil, fmt.Errorf("检查用户数量失败: %w", err)
	}

	// 如果已有用户，不允许注册
	if count > 0 {
		return nil, fmt.Errorf("系统已初始化，不允许注册新用户")
	}

	// 加密密码
	hashedPassword, err := s.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户（第一个用户自动设为管理员）
	user := &db.User{
		Username: req.Username,
		Password: hashedPassword,
		IsAdmin:  true, // 第一个用户自动设为管理员
	}

	if err := s.dbManager.CreateUser(user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 生成token
	token, expiresAt, err := s.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成token失败: %w", err)
	}

	return &LoginResponse{
		Token:     token,
		User:      user,
		ExpiresAt: expiresAt,
	}, nil
}

// GetPublicKey 获取RSA公钥
func (s *AuthService) GetPublicKey() (*PublicKeyResponse, error) {
	// 将公钥转换为PEM格式
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&s.rsaPrivKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("序列化公钥失败: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return &PublicKeyResponse{
		PublicKey: string(pubKeyPEM),
	}, nil
}

// DecryptPassword 解密密码
func (s *AuthService) DecryptPassword(encryptedPassword string) (string, error) {
	// Base64解码
	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败: %w", err)
	}

	// RSA-OAEP解密（匹配前端的加密方式）
	decryptedBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, s.rsaPrivKey, encryptedBytes, nil)
	if err != nil {
		return "", fmt.Errorf("RSA解密失败: %w", err)
	}

	return string(decryptedBytes), nil
}

// EncryptedLogin 加密登录
func (s *AuthService) EncryptedLogin(req *EncryptedLoginRequest) (*LoginResponse, error) {
	// 解密密码
	password, err := s.DecryptPassword(req.EncryptedPassword)
	if err != nil {
		return nil, fmt.Errorf("密码解密失败: %w", err)
	}

	// 使用解密后的密码进行登录
	loginReq := &LoginRequest{
		Username: req.Username,
		Password: password,
	}

	return s.Login(loginReq)
}

// EncryptedRegister 加密注册
func (s *AuthService) EncryptedRegister(req *EncryptedRegisterRequest) (*LoginResponse, error) {
	// 解密密码
	password, err := s.DecryptPassword(req.EncryptedPassword)
	if err != nil {
		return nil, fmt.Errorf("密码解密失败: %w", err)
	}

	// 使用解密后的密码进行注册
	registerReq := &RegisterRequest{
		Username: req.Username,
		Password: password,
	}

	return s.Register(registerReq)
}

// IsFirstInstall 检查是否为首次安装
func (s *AuthService) IsFirstInstall() (bool, error) {
	count, err := s.dbManager.GetUserCount()
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(id uint) (*db.User, error) {
	return s.dbManager.GetUserByID(id)
}

// 用户管理相关结构体

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	IsAdmin  bool   `json:"is_admin"`
}

// CreateUserResponse 创建用户响应
type CreateUserResponse struct {
	User              *db.User `json:"user"`
	GeneratedPassword string   `json:"generated_password"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username  string `json:"username"`
	IsAdmin   *bool  `json:"is_admin"`
	IsEnabled *bool  `json:"is_enabled"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// AdminChangePasswordRequest 管理员修改用户密码请求
type AdminChangePasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Users []UserInfo `json:"users"`
	Total int        `json:"total"`
}

// UserInfo 用户信息（不包含密码）
type UserInfo struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	IsAdmin     bool       `json:"is_admin"`
	IsEnabled   bool       `json:"is_enabled"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedBy   uint       `json:"created_by"`
}

// 用户管理相关方法

// GenerateRandomPassword 生成随机密码
func (s *AuthService) GenerateRandomPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	const length = 12

	password := make([]byte, length)
	for i := range password {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		password[i] = charset[randomByte[0]%byte(len(charset))]
	}

	return string(password)
}

// CreateUser 创建用户（管理员功能）
func (s *AuthService) CreateUser(req *CreateUserRequest, creatorID uint) (*CreateUserResponse, error) {
	// 检查用户名是否已存在
	_, err := s.dbManager.GetUserByUsername(req.Username)
	if err == nil {
		return nil, fmt.Errorf("用户名已存在")
	}

	// 生成随机密码
	password := s.GenerateRandomPassword()
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户
	user := &db.User{
		Username:  req.Username,
		Password:  hashedPassword,
		IsAdmin:   req.IsAdmin,
		IsEnabled: true,
		CreatedBy: creatorID,
	}

	if err := s.dbManager.CreateUser(user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return &CreateUserResponse{
		User:              user,
		GeneratedPassword: password,
	}, nil
}

// GetAllUsers 获取所有用户列表（不包括管理员账号）
func (s *AuthService) GetAllUsers() (*UserListResponse, error) {
	users, err := s.dbManager.GetAllUsers()
	if err != nil {
		return nil, fmt.Errorf("获取用户列表失败: %w", err)
	}

	// 过滤掉管理员账号，只返回普通用户
	var userInfos []UserInfo
	for _, user := range users {
		if !user.IsAdmin {
			userInfos = append(userInfos, UserInfo{
				ID:          user.ID,
				Username:    user.Username,
				IsAdmin:     user.IsAdmin,
				IsEnabled:   user.IsEnabled,
				CreatedAt:   user.CreatedAt,
				UpdatedAt:   user.UpdatedAt,
				LastLoginAt: user.LastLoginAt,
				CreatedBy:   user.CreatedBy,
			})
		}
	}

	return &UserListResponse{
		Users: userInfos,
		Total: len(userInfos),
	}, nil
}

// UpdateUser 更新用户信息
func (s *AuthService) UpdateUser(userID uint, req *UpdateUserRequest) error {
	user, err := s.dbManager.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 更新用户名
	if req.Username != "" && req.Username != user.Username {
		// 检查新用户名是否已存在
		_, err := s.dbManager.GetUserByUsername(req.Username)
		if err == nil {
			return fmt.Errorf("用户名已存在")
		}
		user.Username = req.Username
	}

	// 更新管理员状态
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	// 更新启用状态
	if req.IsEnabled != nil {
		user.IsEnabled = *req.IsEnabled
	}

	return s.dbManager.UpdateUser(user)
}

// DeleteUser 删除用户
func (s *AuthService) DeleteUser(userID uint) error {
	// 检查用户是否存在
	_, err := s.dbManager.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在")
	}

	return s.dbManager.DeleteUser(userID)
}

// ChangePassword 用户修改自己的密码
func (s *AuthService) ChangePassword(userID uint, req *ChangePasswordRequest) error {
	user, err := s.dbManager.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 验证旧密码
	if !s.CheckPassword(user.Password, req.OldPassword) {
		return fmt.Errorf("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := s.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	return s.dbManager.UpdateUserPassword(userID, hashedPassword)
}

// AdminChangePassword 管理员修改用户密码
func (s *AuthService) AdminChangePassword(userID uint, req *AdminChangePasswordRequest) error {
	// 检查用户是否存在
	_, err := s.dbManager.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 加密新密码
	hashedPassword, err := s.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	return s.dbManager.UpdateUserPassword(userID, hashedPassword)
}

// UpdateUserStatus 更新用户状态
func (s *AuthService) UpdateUserStatus(userID uint, isEnabled bool) error {
	return s.dbManager.UpdateUserStatus(userID, isEnabled)
}

// API Key 管理相关方法

// GetAPIKeysByUserID 获取用户的API Key列表
func (s *AuthService) GetAPIKeysByUserID(userID uint) ([]db.APIKey, error) {
	return s.dbManager.GetAPIKeysByUserID(userID)
}

// CreateAPIKey 创建API Key
func (s *AuthService) CreateAPIKey(userID uint, name, keyValue, expiresAt string) (*db.APIKey, error) {
	// 解析过期时间
	var expiresAtTime *time.Time
	if expiresAt != "" {
		parsedTime, err := time.Parse("2006-01-02T15:04:05Z07:00", expiresAt)
		if err != nil {
			return nil, fmt.Errorf("过期时间格式错误: %w", err)
		}
		expiresAtTime = &parsedTime
	}

	// 创建API Key
	apiKey := &db.APIKey{
		UserID:    userID,
		Name:      name,
		KeyValue:  keyValue,
		IsEnabled: true,
		ExpiresAt: expiresAtTime,
	}

	err := s.dbManager.CreateAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("创建API Key失败: %w", err)
	}

	return apiKey, nil
}

// DeleteAPIKey 删除API Key
func (s *AuthService) DeleteAPIKey(apiKeyID, userID uint) error {
	return s.dbManager.DeleteAPIKey(apiKeyID, userID)
}

// GetAPIKeyByValue 根据key值获取API Key
func (s *AuthService) GetAPIKeyByValue(keyValue string) (*db.APIKey, error) {
	return s.dbManager.GetAPIKeyByValue(keyValue)
}

// UpdateAPIKeyLastUsed 更新API Key最后使用时间
func (s *AuthService) UpdateAPIKeyLastUsed(keyValue string) error {
	return s.dbManager.UpdateAPIKeyLastUsed(keyValue)
}
