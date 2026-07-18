package service

import (
	"context"
	"errors"
	"testing"

	"dormmarket/internal/models"
	"dormmarket/internal/repository"
)

func setupReportService(t *testing.T) (*ReportService, *fakeListingRepository, *fakeUserRepository, *fakeReportRepository) {
	t.Helper()
	listings := newFakeListingRepository()
	users := newFakeUserRepository()
	reports := newFakeReportRepository()
	svc := NewReportService(reports, listings, users)
	return svc, listings, users, reports
}

func TestReportService_Create(t *testing.T) {
	t.Run("รายงานประกาศสำเร็จ", func(t *testing.T) {
		svc, listings, _, _ := setupReportService(t)
		ctx := context.Background()

		listing := &models.Listing{SellerID: "seller-1", CategoryID: "cat-1", Title: "โต๊ะ", Price: 100}
		listings.Create(ctx, listing)

		report, err := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetListing,
			TargetListingID: listing.ID, Reason: models.ReportReasonScam,
		})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if report.Status != models.ReportPending {
			t.Errorf("สถานะเริ่มต้นต้องเป็น pending ได้ %s", report.Status)
		}
	})

	t.Run("รายงานผู้ใช้สำเร็จ", func(t *testing.T) {
		svc, _, users, _ := setupReportService(t)
		ctx := context.Background()

		users.Create(ctx, &models.User{Email: "bad@ku.ac.th", Name: "คนไม่ดี"})
		var target *models.User
		for _, u := range users.byID {
			target = u
		}

		_, err := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetUser,
			TargetUserID: target.ID, Reason: models.ReportReasonHarassment,
		})
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
	})

	t.Run("เหตุผลไม่ถูกต้อง", func(t *testing.T) {
		svc, listings, _, _ := setupReportService(t)
		ctx := context.Background()
		listing := &models.Listing{SellerID: "seller-1", Title: "โต๊ะ"}
		listings.Create(ctx, listing)

		_, err := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetListing,
			TargetListingID: listing.ID, Reason: "ไม่มีจริง",
		})
		if !errors.Is(err, ErrInvalidReportReason) {
			t.Fatalf("ต้องการ ErrInvalidReportReason แต่ได้ %v", err)
		}
	})

	t.Run("รายงานตัวเองไม่ได้", func(t *testing.T) {
		svc, _, users, _ := setupReportService(t)
		ctx := context.Background()
		users.Create(ctx, &models.User{Email: "me@ku.ac.th", Name: "ฉันเอง"})
		var me *models.User
		for _, u := range users.byID {
			me = u
		}

		_, err := svc.Create(ctx, CreateReportInput{
			ReporterID: me.ID, TargetType: models.ReportTargetUser,
			TargetUserID: me.ID, Reason: models.ReportReasonOther,
		})
		if !errors.Is(err, ErrCannotReportSelf) {
			t.Fatalf("ต้องการ ErrCannotReportSelf แต่ได้ %v", err)
		}
	})

	t.Run("เป้าหมายไม่มีอยู่จริง", func(t *testing.T) {
		svc, _, _, _ := setupReportService(t)
		ctx := context.Background()

		_, err := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetListing,
			TargetListingID: "ไม่มีจริง", Reason: models.ReportReasonSpam,
		})
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("ต้องการ ErrNotFound แต่ได้ %v", err)
		}
	})
}

func TestReportService_Resolve(t *testing.T) {
	t.Run("dismiss ด้วย action none", func(t *testing.T) {
		svc, listings, _, _ := setupReportService(t)
		ctx := context.Background()
		listing := &models.Listing{SellerID: "seller-1", Title: "โต๊ะ"}
		listings.Create(ctx, listing)
		report, _ := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetListing,
			TargetListingID: listing.ID, Reason: models.ReportReasonOther,
		})

		resolved, err := svc.Resolve(ctx, report.ID, "admin-1", models.ResolutionNone, "ไม่พบปัญหา")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if resolved.Status != models.ReportDismissed {
			t.Errorf("สถานะต้องเป็น dismissed ได้ %s", resolved.Status)
		}
	})

	t.Run("ban_user แบนผู้ใช้ที่ถูกรายงานจริง", func(t *testing.T) {
		svc, _, users, _ := setupReportService(t)
		ctx := context.Background()
		users.Create(ctx, &models.User{Email: "bad@ku.ac.th", Name: "คนไม่ดี"})
		var target *models.User
		for _, u := range users.byID {
			target = u
		}
		report, _ := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetUser,
			TargetUserID: target.ID, Reason: models.ReportReasonHarassment,
		})

		resolved, err := svc.Resolve(ctx, report.ID, "admin-1", models.ResolutionBanUser, "คุกคามผู้ใช้อื่น")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if resolved.Status != models.ReportResolved {
			t.Errorf("สถานะต้องเป็น resolved ได้ %s", resolved.Status)
		}
		if !users.byID[target.ID].IsBanned {
			t.Error("ผู้ใช้ที่ถูกรายงานต้องถูกแบน")
		}
	})

	t.Run("remove_listing ลบประกาศที่ถูกรายงานจริง", func(t *testing.T) {
		svc, listings, _, _ := setupReportService(t)
		ctx := context.Background()
		listing := &models.Listing{SellerID: "seller-1", Title: "ของเถื่อน"}
		listings.Create(ctx, listing)
		report, _ := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetListing,
			TargetListingID: listing.ID, Reason: models.ReportReasonScam,
		})

		_, err := svc.Resolve(ctx, report.ID, "admin-1", models.ResolutionRemoveListing, "ของผิดกฎหมาย")
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		if listings.byID[listing.ID].DeletedAt == nil {
			t.Error("ประกาศที่ถูกรายงานต้องถูกลบ (soft delete)")
		}
	})

	t.Run("ban_user กับ report ประเภท listing ไม่ได้", func(t *testing.T) {
		svc, listings, _, _ := setupReportService(t)
		ctx := context.Background()
		listing := &models.Listing{SellerID: "seller-1", Title: "โต๊ะ"}
		listings.Create(ctx, listing)
		report, _ := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetListing,
			TargetListingID: listing.ID, Reason: models.ReportReasonOther,
		})

		_, err := svc.Resolve(ctx, report.ID, "admin-1", models.ResolutionBanUser, "")
		if !errors.Is(err, ErrResolutionActionMismatch) {
			t.Fatalf("ต้องการ ErrResolutionActionMismatch แต่ได้ %v", err)
		}
	})

	t.Run("resolve ซ้ำไม่ได้", func(t *testing.T) {
		svc, listings, _, _ := setupReportService(t)
		ctx := context.Background()
		listing := &models.Listing{SellerID: "seller-1", Title: "โต๊ะ"}
		listings.Create(ctx, listing)
		report, _ := svc.Create(ctx, CreateReportInput{
			ReporterID: "buyer-1", TargetType: models.ReportTargetListing,
			TargetListingID: listing.ID, Reason: models.ReportReasonOther,
		})
		svc.Resolve(ctx, report.ID, "admin-1", models.ResolutionNone, "")

		_, err := svc.Resolve(ctx, report.ID, "admin-1", models.ResolutionNone, "")
		if !errors.Is(err, ErrReportAlreadyResolved) {
			t.Fatalf("ต้องการ ErrReportAlreadyResolved แต่ได้ %v", err)
		}
	})
}
