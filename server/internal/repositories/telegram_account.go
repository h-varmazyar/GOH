package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/h-varmazyar/gatec/internal/models"
	"github.com/h-varmazyar/gatec/pkg/paginator"
	"gorm.io/gorm"
)

type TelegramAccountRepository struct {
	db *gorm.DB
}

func NewTelegramAccountRepository(db *gorm.DB) *TelegramAccountRepository {
	return &TelegramAccountRepository{db}
}

func (r *TelegramAccountRepository) Upsert(ctx context.Context, in *models.TelegramAccount) error {
	// Validation
	if in.BotToken != nil && in.Mobile != nil && *in.BotToken != "" && *in.Mobile != "" {
		return errors.New("bot_token and mobile cannot both be set")
	}

	if in.BotToken != nil && *in.BotToken != "" {
		return r.upsertByBotToken(ctx, in)
	} else if in.Mobile != nil && *in.Mobile != "" {
		return r.upsertByMobile(ctx, in)
	}

	return errors.New("either bot_token or mobile must be provided")
}

func (r *TelegramAccountRepository) upsertByBotToken(ctx context.Context, in *models.TelegramAccount) error {
	query := `
		INSERT INTO telegram_accounts (
			id, created_at, updated_at, deleted_at,
			role, status, database_directory, files_directory,
			bot_token, mobile, username, account_name, user_id
		)
		VALUES (
			@id, @created_at, @updated_at, @deleted_at,
			@role, @status, @database_directory, @files_directory,
			@bot_token, @mobile, @username, @account_name, @user_id
		)
		ON CONFLICT (bot_token) WHERE bot_token IS NOT NULL
		DO UPDATE SET
			role = EXCLUDED.role,
			database_directory = EXCLUDED.database_directory,
			files_directory = EXCLUDED.files_directory,
			username = EXCLUDED.username,
			account_name = EXCLUDED.account_name,
			user_id = EXCLUDED.user_id,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	return r.db.WithContext(ctx).Exec(query, map[string]interface{}{
		"id":                 in.ID,
		"created_at":         in.CreatedAt,
		"updated_at":         in.UpdatedAt,
		"deleted_at":         in.DeletedAt,
		"role":               in.Role,
		"database_directory": in.DatabaseDirectory,
		"files_directory":    in.FilesDirectory,
		"bot_token":          in.BotToken,
		"mobile":             in.Mobile,
		"username":           in.Username,
		"account_name":       in.AccountName,
		"user_id":            in.UserID,
		"status":             in.Status,
	}).Error
}

func (r *TelegramAccountRepository) upsertByMobile(ctx context.Context, in *models.TelegramAccount) error {
	query := `
		INSERT INTO telegram_accounts (
			id, created_at, updated_at, deleted_at,
			role, status, database_directory, files_directory,
			bot_token, mobile, username, account_name, user_id
		)
		VALUES (
			@id, @created_at, @updated_at, @deleted_at,
			@role, @status, @database_directory, @files_directory,
			@bot_token, @mobile, @username, @account_name, @user_id
		)
		ON CONFLICT (mobile) WHERE mobile IS NOT NULL
		DO UPDATE SET
			role = EXCLUDED.role,
			database_directory = EXCLUDED.database_directory,
			files_directory = EXCLUDED.files_directory,
			username = EXCLUDED.username,
			account_name = EXCLUDED.account_name,
			user_id = EXCLUDED.user_id,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	return r.db.WithContext(ctx).Exec(query, map[string]interface{}{
		"id":                 in.ID,
		"created_at":         in.CreatedAt,
		"updated_at":         in.UpdatedAt,
		"deleted_at":         in.DeletedAt,
		"role":               in.Role,
		"database_directory": in.DatabaseDirectory,
		"files_directory":    in.FilesDirectory,
		"bot_token":          in.BotToken,
		"mobile":             in.Mobile,
		"username":           in.Username,
		"account_name":       in.AccountName,
		"user_id":            in.UserID,
		"status":             in.Status,
	}).Error
}

// func (r *TelegramAccountRepository) Upsert(ctx context.Context, in *models.TelegramAccount) error {
// 	// if (in.BotToken != nil  && *in.BotToken != "")||(in.Mobile != nil && *in.Mobile != "") {
// 	// 	return errors.New("bot_token and mobile cannot both be set")
// 	// }

// 	var onConflict clause.OnConflict

// 	if in.BotToken != nil && *in.BotToken != "" {
// 		// اکانت Bot: بر اساس bot_token چک شود
// 		onConflict = clause.OnConflict{
// 			OnConstraint: "idx_unique_bot_token",
// 			Where:        clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "bot_token IS NOT NULL"}}},
// 			DoUpdates: clause.AssignmentColumns([]string{
// 				"role",
// 				"database_directory",
// 				"files_directory",
// 				"account_name",
// 				"updated_at",
// 			}),
// 		}
// 	} else if in.Mobile != nil && *in.Mobile != "" {
// 		// اکانت User: بر اساس mobile چک شود
// 		onConflict = clause.OnConflict{
// 			OnConstraint: "idx_unique_mobile",
// 			Where:        clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "mobile IS NOT NULL"}}},
// 			DoUpdates: clause.AssignmentColumns([]string{
// 				"role",
// 				"database_directory",
// 				"files_directory",
// 				"username",
// 				"account_name",
// 				"updated_at",
// 			}),
// 		}
// 	} else {
// 		return errors.New("either bot_token or mobile must be provided")
// 	}

// 	return r.db.WithContext(ctx).Clauses(onConflict).Create(in).Error
// }

func (r *TelegramAccountRepository) Get(ctx context.Context, conditions map[string]any) (*models.TelegramAccount, error) {
	account := new(models.TelegramAccount)

	tx := r.db.WithContext(ctx).Model(&models.TelegramAccount{})
	for condition, value := range conditions {
		tx.Where(condition, value)
	}

	return account, tx.First(account).Error
}

func (r *TelegramAccountRepository) List(ctx context.Context, pg paginator.Params, conditions map[string]any) ([]models.TelegramAccount, int64, error) {
	telegramAccounts := make([]models.TelegramAccount, 0)
	var total int64

	tx := r.db.WithContext(ctx).Model(&models.TelegramAccount{})
	for condition, value := range conditions {
		tx = tx.Where(condition, value)
	}

	tx.Count(&total)

	tx.Find(&telegramAccounts)
	fmt.Println("s1:", len(telegramAccounts), pg.Offset(), pg.Limit())

	err := pg.Apply(tx).Find(&telegramAccounts).Error

	return telegramAccounts, total, err
}

func (r *TelegramAccountRepository) UpdateStatus(ctx context.Context, id string, status models.TelegramAccountStatus) error {
	return r.db.WithContext(ctx).
		Model(&models.TelegramAccount{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *TelegramAccountRepository) ListAll(ctx context.Context) ([]models.TelegramAccount, error) {
	telegramAccounts := make([]models.TelegramAccount, 0)
	return telegramAccounts, r.db.WithContext(ctx).Find(&telegramAccounts).Error
}
