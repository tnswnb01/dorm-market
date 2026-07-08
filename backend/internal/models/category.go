package models

// Category คือหมวดหมู่สินค้า เช่น หนังสือ, เฟอร์นิเจอร์, เครื่องใช้ไฟฟ้า
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
