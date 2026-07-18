package models

import "time"

type ReportTargetType string

const (
	ReportTargetListing ReportTargetType = "listing"
	ReportTargetUser    ReportTargetType = "user"
)

type ReportReason string

const (
	ReportReasonScam          ReportReason = "scam"
	ReportReasonInappropriate ReportReason = "inappropriate"
	ReportReasonHarassment    ReportReason = "harassment"
	ReportReasonSpam          ReportReason = "spam"
	ReportReasonOther         ReportReason = "other"
)

type ReportStatus string

const (
	ReportPending   ReportStatus = "pending"
	ReportResolved  ReportStatus = "resolved"
	ReportDismissed ReportStatus = "dismissed"
)

type ReportResolutionAction string

const (
	ResolutionBanUser       ReportResolutionAction = "ban_user"
	ResolutionRemoveListing ReportResolutionAction = "remove_listing"
	ResolutionNone          ReportResolutionAction = "none"
)

// Report คือรายงานที่ผู้ใช้แจ้งเข้ามา — รายงานได้ทั้งประกาศ (listing) และผู้ใช้ (user)
// มีอย่างใดอย่างหนึ่งเท่านั้นระหว่าง TargetListingID/TargetUserID ตาม TargetType (บังคับด้วย DB CHECK)
type Report struct {
	ID               string                  `json:"id"`
	ReporterID       string                  `json:"reporterId"`
	Reporter         *PublicUser             `json:"reporter,omitempty"`
	TargetType       ReportTargetType        `json:"targetType"`
	TargetListingID  *string                 `json:"targetListingId,omitempty"`
	TargetListing    *Listing                `json:"targetListing,omitempty"`
	TargetUserID     *string                 `json:"targetUserId,omitempty"`
	TargetUser       *PublicUser             `json:"targetUser,omitempty"`
	Reason           ReportReason            `json:"reason"`
	Description      string                  `json:"description"`
	Status           ReportStatus            `json:"status"`
	ResolutionAction *ReportResolutionAction `json:"resolutionAction,omitempty"`
	ResolutionNote   string                  `json:"resolutionNote,omitempty"`
	ResolvedBy       *string                 `json:"resolvedBy,omitempty"`
	ResolvedAt       *time.Time              `json:"resolvedAt,omitempty"`
	CreatedAt        time.Time               `json:"createdAt"`
}
