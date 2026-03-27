package models

import "gorm.io/gorm"

// PasswordEntry 密码条目
type PasswordEntry struct {
	gorm.Model
	Title      string    `json:"title"`
	Username   string    `json:"username"`
	Password   string    `json:"-"`
	URL        string    `json:"url"`
	CategoryID *uint     `json:"categoryId"`
	Category   *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	Tags       []Tag     `json:"tags" gorm:"many2many:entry_tags;"`
	Notes      string    `json:"notes"`
}

// MasterKey 主密钥信息
type MasterKey struct {
	gorm.Model
	Salt     string `json:"-"`
	Verifier string `json:"-"`
}

// Category 树形分类
type Category struct {
	gorm.Model
	Name     string     `json:"name"`
	ParentID *uint      `json:"parentId"`
	Parent   *Category  `json:"-" gorm:"foreignKey:ParentID"`
	Children []Category `json:"children" gorm:"-"`
	Icon     string     `json:"icon"`
	Order    int        `json:"order"`
}

// Tag 标签
type Tag struct {
	gorm.Model
	Name  string `json:"name"`
	Color string `json:"color"`
}

// TagDTO 标签传输对象
type TagDTO struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// CategoryDTO 分类传输对象（含树形 children）
type CategoryDTO struct {
	ID       uint           `json:"id"`
	Name     string         `json:"name"`
	ParentID *uint          `json:"parentId"`
	Icon     string         `json:"icon"`
	Order    int            `json:"order"`
	Children []*CategoryDTO `json:"children"`
}

// EntryDTO 前端交互的数据传输对象
type EntryDTO struct {
	ID          uint     `json:"id"`
	Title       string   `json:"title"`
	Username    string   `json:"username"`
	Password    string   `json:"password,omitempty"`
	URL         string   `json:"url"`
	CategoryID  *uint    `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	TagIDs      []uint   `json:"tagIds"`
	Tags        []TagDTO `json:"tags"`
	Notes       string   `json:"notes"`
}

// ToDTO 转换为 DTO (不含密码)
func (e *PasswordEntry) ToDTO() *EntryDTO {
	dto := &EntryDTO{
		ID:          e.ID,
		Title:       e.Title,
		Username:    e.Username,
		URL:         e.URL,
		CategoryID:  e.CategoryID,
		Notes:       e.Notes,
	}
	if e.Category != nil {
		dto.CategoryName = e.Category.Name
	}
	for _, t := range e.Tags {
		dto.Tags = append(dto.Tags, TagDTO{ID: t.ID, Name: t.Name, Color: t.Color})
		dto.TagIDs = append(dto.TagIDs, t.ID)
	}
	return dto
}

// ToDTO 转换 Category 为 DTO
func (c *Category) ToDTO() *CategoryDTO {
	return &CategoryDTO{
		ID:       c.ID,
		Name:     c.Name,
		ParentID: c.ParentID,
		Icon:     c.Icon,
		Order:    c.Order,
	}
}
