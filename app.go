package main

import (
	"else-toolbox/internal/crypto"
	"else-toolbox/internal/database"
	"else-toolbox/internal/models"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx       context.Context
	dataDir   string
	masterKey []byte
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	homeDir, _ := os.UserHomeDir()
	a.dataDir = filepath.Join(homeDir, ".else-toolbox")
	os.MkdirAll(a.dataDir, 0700)

	database.Init(a.dataDir)
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	database.Close()
}

// IsInitialized 检查是否已设置主密码
func (a *App) IsInitialized() bool {
	var count int64
	database.DB.Model(&models.MasterKey{}).Count(&count)
	return count > 0
}

// SetupMasterKey 设置主密码（首次使用）
func (a *App) SetupMasterKey(password string) error {
	if a.IsInitialized() {
		return errors.New("master key already initialized")
	}

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return err
	}

	key := crypto.DeriveKey([]byte(password), salt)

	verifier, err := crypto.Encrypt([]byte("vault-verified"), key)
	if err != nil {
		return err
	}

	masterKey := models.MasterKey{
		Salt:     base64.StdEncoding.EncodeToString(salt),
		Verifier: verifier,
	}

	a.masterKey = key
	return database.DB.Create(&masterKey).Error
}

// Unlock 使用主密码解锁
func (a *App) Unlock(password string) bool {
	var mk models.MasterKey
	if err := database.DB.First(&mk).Error; err != nil {
		return false
	}

	salt, _ := base64.StdEncoding.DecodeString(mk.Salt)
	key := crypto.DeriveKey([]byte(password), salt)

	_, err := crypto.Decrypt(mk.Verifier, key)
	if err != nil {
		return false
	}

	a.masterKey = key
	return true
}

// IsUnlocked 检查是否已解锁
func (a *App) IsUnlocked() bool {
	return a.masterKey != nil
}

// Lock 锁定
func (a *App) Lock() {
	a.masterKey = nil
}

// ==================== Entry CRUD ====================

// GetEntries 获取密码条目，可按分类或标签过滤
func (a *App) GetEntries(categoryID *uint, tagID *uint) ([]*models.EntryDTO, error) {
	if !a.IsUnlocked() {
		return nil, errors.New("vault is locked")
	}

	query := database.DB.Model(&models.PasswordEntry{})
	if categoryID != nil {
		query = query.Where("category_id = ?", *categoryID)
	}
	if tagID != nil {
		query = query.Joins("JOIN entry_tags ON entry_tags.password_entry_id = password_entries.id AND entry_tags.tag_id = ?", *tagID)
	}

	var entries []*models.PasswordEntry
	if err := query.Preload("Category").Preload("Tags").Find(&entries).Error; err != nil {
		return nil, err
	}

	result := make([]*models.EntryDTO, len(entries))
	for i, e := range entries {
		result[i] = e.ToDTO()
	}
	return result, nil
}

// CreateEntry 创建密码条目
func (a *App) CreateEntry(dto models.EntryDTO) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}

	encryptedPwd, err := crypto.Encrypt([]byte(dto.Password), a.masterKey)
	if err != nil {
		return err
	}

	entry := models.PasswordEntry{
		Title:     dto.Title,
		Username:  dto.Username,
		Password:  encryptedPwd,
		URL:       dto.URL,
		Notes:     dto.Notes,
	}

	if dto.CategoryID != nil {
		entry.CategoryID = dto.CategoryID
	}

	if len(dto.TagIDs) > 0 {
		var tags []models.Tag
		database.DB.Where("id IN ?", dto.TagIDs).Find(&tags)
		entry.Tags = tags
	}

	return database.DB.Create(&entry).Error
}

// UpdateEntry 更新密码条目
func (a *App) UpdateEntry(dto models.EntryDTO) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}

	var entry models.PasswordEntry
	if err := database.DB.First(&entry, dto.ID).Error; err != nil {
		return err
	}

	entry.Title = dto.Title
	entry.Username = dto.Username
	entry.URL = dto.URL
	entry.Notes = dto.Notes
	entry.CategoryID = dto.CategoryID

	if dto.Password != "" {
		encryptedPwd, err := crypto.Encrypt([]byte(dto.Password), a.masterKey)
		if err != nil {
			return err
		}
		entry.Password = encryptedPwd
	}

	// 更新标签关联
	if len(dto.TagIDs) > 0 {
		var tags []models.Tag
		database.DB.Where("id IN ?", dto.TagIDs).Find(&tags)
		entry.Tags = tags
	} else {
		entry.Tags = nil
	}

	return database.DB.Save(&entry).Error
}

// DeleteEntry 删除密码条目
func (a *App) DeleteEntry(id uint) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}
	return database.DB.Delete(&models.PasswordEntry{}, id).Error
}

// GetPassword 获取解密后的密码
func (a *App) GetPassword(id uint) (string, error) {
	if !a.IsUnlocked() {
		return "", errors.New("vault is locked")
	}

	var entry models.PasswordEntry
	if err := database.DB.First(&entry, id).Error; err != nil {
		return "", err
	}

	decrypted, err := crypto.Decrypt(entry.Password, a.masterKey)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// ==================== Category CRUD ====================

// GetCategoryTree 获取树形分类结构
func (a *App) GetCategoryTree() ([]*models.CategoryDTO, error) {
	if !a.IsUnlocked() {
		return nil, errors.New("vault is locked")
	}

	var categories []models.Category
	if err := database.DB.Order("`order` ASC, id ASC").Find(&categories).Error; err != nil {
		return nil, err
	}

	// 构建树
	dtoMap := make(map[uint]*models.CategoryDTO)
	for _, c := range categories {
		dto := c.ToDTO()
		dto.Children = []*models.CategoryDTO{}
		dtoMap[c.ID] = dto
	}

	var roots []*models.CategoryDTO
	for _, c := range categories {
		dto := dtoMap[c.ID]
		if c.ParentID == nil {
			roots = append(roots, dto)
		} else if parent, ok := dtoMap[*c.ParentID]; ok {
			parent.Children = append(parent.Children, dto)
		} else {
			roots = append(roots, dto)
		}
	}

	return roots, nil
}

// CreateCategory 创建分类
func (a *App) CreateCategory(name string, parentID *uint) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}
	cat := models.Category{Name: name, ParentID: parentID}
	return database.DB.Create(&cat).Error
}

// UpdateCategory 更新分类
func (a *App) UpdateCategory(id uint, name string, parentID *uint) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}

	var cat models.Category
	if err := database.DB.First(&cat, id).Error; err != nil {
		return err
	}

	cat.Name = name
	cat.ParentID = parentID

	// 防止循环引用：不能把自己设为子节点
	if parentID != nil && *parentID == id {
		return errors.New("不能将分类设为自己的子分类")
	}

	return database.DB.Save(&cat).Error
}

// DeleteCategory 删除分类
func (a *App) DeleteCategory(id uint) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}

	// 将子分类移到父级
	database.DB.Model(&models.Category{}).Where("parent_id = ?", id).Update("parent_id", nil)

	// 清除条目的分类关联
	database.DB.Model(&models.PasswordEntry{}).Where("category_id = ?", id).Update("category_id", nil)

	return database.DB.Delete(&models.Category{}, id).Error
}

// ==================== Tag CRUD ====================

// GetTags 获取所有标签
func (a *App) GetTags() ([]*models.TagDTO, error) {
	if !a.IsUnlocked() {
		return nil, errors.New("vault is locked")
	}

	var tags []models.Tag
	if err := database.DB.Find(&tags).Error; err != nil {
		return nil, err
	}

	result := make([]*models.TagDTO, len(tags))
	for i, t := range tags {
		result[i] = &models.TagDTO{ID: t.ID, Name: t.Name, Color: t.Color}
	}
	return result, nil
}

// CreateTag 创建标签
func (a *App) CreateTag(name string, color string) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}
	tag := models.Tag{Name: name, Color: color}
	return database.DB.Create(&tag).Error
}

// UpdateTag 更新标签
func (a *App) UpdateTag(id uint, name string, color string) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}

	var tag models.Tag
	if err := database.DB.First(&tag, id).Error; err != nil {
		return err
	}

	tag.Name = name
	tag.Color = color
	return database.DB.Save(&tag).Error
}

// DeleteTag 删除标签
func (a *App) DeleteTag(id uint) error {
	if !a.IsUnlocked() {
		return errors.New("vault is locked")
	}
	return database.DB.Delete(&models.Tag{}, id).Error
}

// ==================== Windows Hello ====================

// SetupHello stores the master key encrypted with DPAPI for Windows Hello unlock
func (a *App) SetupHello() error {
	if a.masterKey == nil {
		return errors.New("vault is locked")
	}
	encrypted, err := crypto.DPAPIEncrypt(a.masterKey)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.dataDir, ".hello_key"), encrypted, 0600)
}

// IsHelloEnabled checks if Windows Hello unlock is configured
func (a *App) IsHelloEnabled() bool {
	_, err := os.Stat(filepath.Join(a.dataDir, ".hello_key"))
	return err == nil
}

// DisableHello removes Windows Hello configuration
func (a *App) DisableHello() error {
	os.Remove(filepath.Join(a.dataDir, ".hello_key"))
	os.Remove(filepath.Join(a.dataDir, ".hello_cred"))
	return nil
}

// StoreHelloCredential stores the WebAuthn credential ID
func (a *App) StoreHelloCredential(credId []byte) error {
	return os.WriteFile(filepath.Join(a.dataDir, ".hello_cred"), credId, 0600)
}

// GetHelloCredential retrieves the stored WebAuthn credential ID
func (a *App) GetHelloCredential() ([]byte, error) {
	return os.ReadFile(filepath.Join(a.dataDir, ".hello_cred"))
}

// UnlockWithHello decrypts the stored master key via DPAPI
func (a *App) UnlockWithHello() bool {
	data, err := os.ReadFile(filepath.Join(a.dataDir, ".hello_key"))
	if err != nil {
		return false
	}
	key, err := crypto.DPAPIDecrypt(data)
	if err != nil {
		return false
	}
	// Verify by decrypting the verifier
	var mk models.MasterKey
	if err := database.DB.First(&mk).Error; err != nil {
		return false
	}
	_, err = crypto.Decrypt(mk.Verifier, key)
	if err != nil {
		return false
	}
	a.masterKey = key
	return true
}

// ==================== Utility ====================

// SelectDirectory 选择目录（用于导出等）
func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择目录",
	})
}
