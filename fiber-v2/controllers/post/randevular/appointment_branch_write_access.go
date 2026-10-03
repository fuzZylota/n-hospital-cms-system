package randevular

import (
	"context"
	"database/sql"
	"errors"
)

// Randevu yazma yollarında (oluşturma, düzenleme, talep durumu) ortak şube yetkisi.
//
// Karar (2026-10-03, kullanıcı): admin tüm şubelerde; moderatör ve santral yalnız
// kendi şubelerinde yazabilir. "Kendi şubesi" okuma kuralıyla aynıdır:
// user_branch_permissions.can_view = true; santral için ayrıca users.sid eşleşmesi.
// user_branch_permissions tablosunda can_view ve can_delete dışında bayrak yoktur.
// Kesinleşmiş randevu SİLME bu kuralın dışındadır ve admin'e özel kalır.
//
// Çağıran, kullanıcı satırını aynı transaction'da FOR UPDATE ile okuyup role ve
// is_active değerini doğrulamış olmalıdır. İstemciden gelen şube kimliği tek
// başına yetki sayılmaz; branchSID kilitli nesnenin gerçek şubesidir.
func isAppointmentWriteRole(role string) bool {
	return role == "admin" || role == "moderator" || role == "santral"
}

// 0: izinli, 403: yetkisiz, 503: okuma hatası. Admin için şubesiz (NULL) hedef de izinlidir.
func authorizeBranchWrite(ctx context.Context, tx *sql.Tx, userID int64, role string, branchSID sql.NullInt64) int {
	if role == "admin" {
		return 0
	}
	if role != "moderator" && role != "santral" {
		return 403
	}
	if !branchSID.Valid || branchSID.Int64 <= 0 {
		return 403
	}
	if role == "santral" {
		var userSID sql.NullInt64
		err := tx.QueryRowContext(ctx, "SELECT sid FROM users WHERE uid = $1 FOR UPDATE", userID).Scan(&userSID)
		if errors.Is(err, sql.ErrNoRows) {
			return 403
		}
		if err != nil {
			return 503
		}
		if !userSID.Valid || userSID.Int64 != branchSID.Int64 {
			return 403
		}
	}
	var canView sql.NullBool
	err := tx.QueryRowContext(ctx, "SELECT can_view FROM user_branch_permissions WHERE uid = $1 AND sid = $2 FOR UPDATE", userID, branchSID.Int64).Scan(&canView)
	if errors.Is(err, sql.ErrNoRows) {
		return 403
	}
	if err != nil {
		return 503
	}
	if !canView.Valid || !canView.Bool {
		return 403
	}
	return 0
}
