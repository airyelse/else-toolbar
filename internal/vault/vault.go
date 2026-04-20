package vault

import (
	"else-toolbox/internal/crypto"
	"else-toolbox/internal/database"
	"else-toolbox/internal/hello"
	"else-toolbox/internal/models"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
)

type Vault struct {
	dataDir   string
	masterKey []byte
}

func New(dataDir string) *Vault {
	return &Vault{dataDir: dataDir}
}

func (v *Vault) IsInitialized() bool {
	var count int64
	database.DB.Model(&models.MasterKey{}).Count(&count)
	return count > 0
}

func (v *Vault) SetupMasterKey(password string) error {
	if v.IsInitialized() {
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

	v.masterKey = key
	return database.DB.Create(&masterKey).Error
}

func (v *Vault) Unlock(password string) (bool, error) {
	var mk models.MasterKey
	if err := database.DB.First(&mk).Error; err != nil {
		return false, err
	}

	salt, _ := base64.StdEncoding.DecodeString(mk.Salt)
	key := crypto.DeriveKey([]byte(password), salt)

	_, err := crypto.Decrypt(mk.Verifier, key)
	if err != nil {
		return false, nil
	}

	v.masterKey = key
	return true, nil
}

func (v *Vault) IsUnlocked() bool {
	return v.masterKey != nil
}

func (v *Vault) Lock() {
	if v.masterKey != nil {
		for i := range v.masterKey {
			v.masterKey[i] = 0
		}
	}
	v.masterKey = nil
}

// ==================== Entry CRUD ====================

func (v *Vault) GetEntries(categoryID *uint, tagID *uint) ([]*models.EntryDTO, error) {
	if !v.IsUnlocked() {
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

func (v *Vault) CreateEntry(dto models.EntryDTO) error {
	if !v.IsUnlocked() {
		return errors.New("vault is locked")
	}

	encryptedPwd, err := crypto.Encrypt([]byte(dto.Password), v.masterKey)
	if err != nil {
		return err
	}

	entry := models.PasswordEntry{
		Title:    dto.Title,
		Username: dto.Username,
		Password: encryptedPwd,
		URL:      dto.URL,
		Notes:    dto.Notes,
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

func (v *Vault) UpdateEntry(dto models.EntryDTO) error {
	if !v.IsUnlocked() {
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
		encryptedPwd, err := crypto.Encrypt([]byte(dto.Password), v.masterKey)
		if err != nil {
			return err
		}
		entry.Password = encryptedPwd
	}

	if len(dto.TagIDs) > 0 {
		var tags []models.Tag
		database.DB.Where("id IN ?", dto.TagIDs).Find(&tags)
		entry.Tags = tags
	} else {
		entry.Tags = nil
	}

	return database.DB.Save(&entry).Error
}

func (v *Vault) DeleteEntry(id uint) error {
	if !v.IsUnlocked() {
		return errors.New("vault is locked")
	}
	return database.DB.Delete(&models.PasswordEntry{}, id).Error
}

func (v *Vault) GetPassword(id uint) (string, error) {
	if !v.IsUnlocked() {
		return "", errors.New("vault is locked")
	}

	var entry models.PasswordEntry
	if err := database.DB.First(&entry, id).Error; err != nil {
		return "", err
	}

	decrypted, err := crypto.Decrypt(entry.Password, v.masterKey)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// ==================== Category CRUD ====================

func (v *Vault) GetCategoryTree() ([]*models.CategoryDTO, error) {
	if !v.IsUnlocked() {
		return nil, errors.New("vault is locked")
	}

	var categories []models.Category
	if err := database.DB.Order("`order` ASC, id ASC").Find(&categories).Error; err != nil {
		return nil, err
	}

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

func (v *Vault) CreateCategory(name string, parentID *uint) error {
	if !v.IsUnlocked() {
		return errors.New("vault is locked")
	}
	cat := models.Category{Name: name, ParentID: parentID}
	return database.DB.Create(&cat).Error
}

func (v *Vault) UpdateCategory(id uint, name string, parentID *uint) error {
	if !v.IsUnlocked() {
		return errors.New("vault is locked")
	}

	// Self-parent check
	if parentID != nil && *parentID == id {
		return errors.New("不能将分类设为自己的子分类")
	}

	// Cycle detection: check if parentID is a descendant of id
	if parentID != nil && isDescendantOf(*parentID, id) {
		return errors.New("不能将分类设为自身子分类的子分类（会形成循环）")
	}

	var cat models.Category
	if err := database.DB.First(&cat, id).Error; err != nil {
		return err
	}

	cat.Name = name
	cat.ParentID = parentID

	return database.DB.Save(&cat).Error
}

// isDescendantOf checks whether categoryID is a descendant of potentialAncestorID
// by walking the ParentID chain upward.
func isDescendantOf(categoryID uint, potentialAncestorID uint) bool {
	current := categoryID
	for {
		var cat models.Category
		if err := database.DB.Select("parent_id").First(&cat, current).Error; err != nil {
			return false
		}
		if cat.ParentID == nil {
			return false
		}
		if *cat.ParentID == potentialAncestorID {
			return true
		}
		current = *cat.ParentID
	}
}

func (v *Vault) DeleteCategory(id uint) error {
	if !v.IsUnlocked() {
		return errors.New("vault is locked")
	}

	database.DB.Model(&models.Category{}).Where("parent_id = ?", id).Update("parent_id", nil)
	database.DB.Model(&models.PasswordEntry{}).Where("category_id = ?", id).Update("category_id", nil)

	return database.DB.Delete(&models.Category{}, id).Error
}

// ==================== Tag CRUD ====================

func (v *Vault) GetTags() ([]*models.TagDTO, error) {
	if !v.IsUnlocked() {
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

func (v *Vault) CreateTag(name string, color string) error {
	if !v.IsUnlocked() {
		return errors.New("vault is locked")
	}
	tag := models.Tag{Name: name, Color: color}
	return database.DB.Create(&tag).Error
}

func (v *Vault) UpdateTag(id uint, name string, color string) error {
	if !v.IsUnlocked() {
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

func (v *Vault) DeleteTag(id uint) error {
	if !v.IsUnlocked() {
		return errors.New("vault is locked")
	}
	return database.DB.Delete(&models.Tag{}, id).Error
}

// ==================== Windows Hello ====================

func (v *Vault) SetupHello() error {
	if v.masterKey == nil {
		return errors.New("vault is locked")
	}
	result, err := hello.RequestVerification("启用 Windows Hello 解锁")
	if err != nil {
		return err
	}
	if result != "Verified" {
		return errors.New("windows hello verification was not completed")
	}
	encrypted, err := crypto.DPAPIEncrypt(v.masterKey)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(v.dataDir, ".hello_key"), encrypted, 0600)
}

func (v *Vault) GetHelloAvailability() string {
	result, err := hello.CheckAvailability()
	if err != nil {
		return "Unknown"
	}
	return result
}

func (v *Vault) IsHelloEnabled() bool {
	_, err := os.Stat(filepath.Join(v.dataDir, ".hello_key"))
	return err == nil
}

func (v *Vault) DisableHello() error {
	os.Remove(filepath.Join(v.dataDir, ".hello_key"))
	os.Remove(filepath.Join(v.dataDir, ".hello_cred"))
	return nil
}

func (v *Vault) StoreHelloCredential(credId []byte) error {
	return os.WriteFile(filepath.Join(v.dataDir, ".hello_cred"), credId, 0600)
}

func (v *Vault) GetHelloCredential() ([]byte, error) {
	return os.ReadFile(filepath.Join(v.dataDir, ".hello_cred"))
}

func (v *Vault) UnlockWithHello() (bool, error) {
	result, err := hello.RequestVerification("使用 Windows Hello 解锁密码库")
	if err != nil {
		return false, err
	}
	if result != "Verified" {
		return false, nil
	}

	data, err := os.ReadFile(filepath.Join(v.dataDir, ".hello_key"))
	if err != nil {
		return false, err
	}
	key, err := crypto.DPAPIDecrypt(data)
	if err != nil {
		return false, err
	}

	var mk models.MasterKey
	if err := database.DB.First(&mk).Error; err != nil {
		return false, err
	}
	_, err = crypto.Decrypt(mk.Verifier, key)
	if err != nil {
		return false, err
	}
	v.masterKey = key
	return true, nil
}

func (v *Vault) OpenWindowsHelloSettings() error {
	return hello.OpenSettings()
}
