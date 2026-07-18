package service

import (
	"context"
	"errors"
	"strings"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

var (
	ErrInvalidReportReason       = errors.New("เหตุผลการรายงานไม่ถูกต้อง")
	ErrInvalidTargetType         = errors.New("ประเภทเป้าหมายการรายงานไม่ถูกต้อง")
	ErrCannotReportSelf          = errors.New("รายงานตัวเองไม่ได้")
	ErrReportAlreadyResolved     = errors.New("รายงานนี้ถูกดำเนินการไปแล้ว")
	ErrInvalidResolutionAction   = errors.New("การดำเนินการไม่ถูกต้อง")
	ErrResolutionActionMismatch  = errors.New("การดำเนินการไม่ตรงกับประเภทของรายงาน")
)

var validReportReasons = map[models.ReportReason]bool{
	models.ReportReasonScam:          true,
	models.ReportReasonInappropriate: true,
	models.ReportReasonHarassment:    true,
	models.ReportReasonSpam:          true,
	models.ReportReasonOther:         true,
}

type ReportService struct {
	reports  repository.ReportRepository
	listings repository.ListingRepository
	users    repository.UserRepository
}

func NewReportService(reports repository.ReportRepository, listings repository.ListingRepository, users repository.UserRepository) *ReportService {
	return &ReportService{reports: reports, listings: listings, users: users}
}

type CreateReportInput struct {
	ReporterID      string
	TargetType      models.ReportTargetType
	TargetListingID string // ใช้เมื่อ TargetType == listing
	TargetUserID    string // ใช้เมื่อ TargetType == user
	Reason          models.ReportReason
	Description     string
}

// Create สร้างรายงานใหม่ หลังตรวจสอบ:
//  1. reason ต้องอยู่ในชุดที่กำหนดไว้
//  2. targetType ต้องเป็น listing หรือ user เท่านั้น และมี target ID ตรงกับ type
//  3. เป้าหมายต้องมีอยู่จริง
//  4. รายงานตัวเองไม่ได้ (กรณี targetType == user)
func (s *ReportService) Create(ctx context.Context, in CreateReportInput) (*models.Report, error) {
	if !validReportReasons[in.Reason] {
		return nil, ErrInvalidReportReason
	}

	report := &models.Report{
		ReporterID:  in.ReporterID,
		TargetType:  in.TargetType,
		Reason:      in.Reason,
		Description: strings.TrimSpace(in.Description),
	}

	switch in.TargetType {
	case models.ReportTargetListing:
		if in.TargetListingID == "" {
			return nil, ErrInvalidTargetType
		}
		if _, err := s.listings.GetByID(ctx, in.TargetListingID); err != nil {
			return nil, err
		}
		report.TargetListingID = &in.TargetListingID
	case models.ReportTargetUser:
		if in.TargetUserID == "" {
			return nil, ErrInvalidTargetType
		}
		if in.TargetUserID == in.ReporterID {
			return nil, ErrCannotReportSelf
		}
		if _, err := s.users.GetByID(ctx, in.TargetUserID); err != nil {
			return nil, err
		}
		report.TargetUserID = &in.TargetUserID
	default:
		return nil, ErrInvalidTargetType
	}

	if err := s.reports.Create(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

// List คืนรายงานทั้งหมด (แอดมินเท่านั้น) — status เป็น nil แปลว่าเอาทุกสถานะ
func (s *ReportService) List(ctx context.Context, status *models.ReportStatus) ([]models.Report, error) {
	return s.reports.List(ctx, status)
}

// Resolve คือขั้นตอนที่แอดมินดำเนินการกับรายงานหนึ่งรายการ จากหน้าเดียว:
//   - action = "none"            -> ปิด report โดยไม่ทำอะไรเพิ่ม (สถานะ dismissed)
//   - action = "ban_user"        -> แบนผู้ใช้ที่ถูกรายงาน (ต้องเป็น report ประเภท user)
//   - action = "remove_listing"  -> ลบประกาศที่ถูกรายงาน แบบ soft delete (ต้องเป็น report ประเภท listing)
func (s *ReportService) Resolve(ctx context.Context, reportID, adminID string, action models.ReportResolutionAction, note string) (*models.Report, error) {
	report, err := s.reports.GetByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	if report.Status != models.ReportPending {
		return nil, ErrReportAlreadyResolved
	}

	status := models.ReportResolved
	switch action {
	case models.ResolutionNone:
		status = models.ReportDismissed
	case models.ResolutionBanUser:
		if report.TargetType != models.ReportTargetUser || report.TargetUserID == nil {
			return nil, ErrResolutionActionMismatch
		}
		if err := s.users.Ban(ctx, *report.TargetUserID, note, adminID); err != nil {
			return nil, err
		}
	case models.ResolutionRemoveListing:
		if report.TargetType != models.ReportTargetListing || report.TargetListingID == nil {
			return nil, ErrResolutionActionMismatch
		}
		if err := s.listings.AdminSoftDelete(ctx, *report.TargetListingID); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
	default:
		return nil, ErrInvalidResolutionAction
	}

	if err := s.reports.Resolve(ctx, reportID, status, action, strings.TrimSpace(note), adminID); err != nil {
		return nil, err
	}
	return s.reports.GetByID(ctx, reportID)
}
