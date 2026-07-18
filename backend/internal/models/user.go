package models

import "time"

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// User คือบัญชีผู้ใช้ในระบบ
type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // ไม่ส่งกลับใน JSON เด็ดขาด
	GoogleID     *string    `json:"-"` // nil = สมัครด้วยอีเมล/รหัสผ่าน, ไม่ nil = ผูกกับบัญชี Google แล้ว
	Name         string     `json:"name"`
	DormBuilding string     `json:"dormBuilding"`
	AvatarURL    string     `json:"avatarUrl"`
	TrustScore   int        `json:"trustScore"`
	Role         UserRole   `json:"role"`
	IsBanned     bool       `json:"isBanned"`
	BanReason    string     `json:"banReason,omitempty"`
	BannedAt     *time.Time `json:"bannedAt,omitempty"`
	BannedBy     *string    `json:"-"` // internal only — ไม่ต้องให้ frontend เห็นว่าแอดมินคนไหนแบน
	CreatedAt    time.Time  `json:"createdAt"`
}

// PublicUser คือข้อมูล user แบบย่อที่ปลอดภัยจะแสดงให้คนอื่นเห็น (เช่น ในหน้ารายละเอียดสินค้า)
type PublicUser struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DormBuilding string `json:"dormBuilding"`
	AvatarURL    string `json:"avatarUrl"`
	TrustScore   int    `json:"trustScore"`
}

func (u User) ToPublic() PublicUser {
	return PublicUser{
		ID:           u.ID,
		Name:         u.Name,
		DormBuilding: u.DormBuilding,
		AvatarURL:    u.AvatarURL,
		TrustScore:   u.TrustScore,
	}
}
