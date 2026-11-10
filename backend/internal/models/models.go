package models

import (
	"time"

	"github.com/google/uuid"
)

type Household struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	DefaultSplitParts []int     `json:"default_split_parts"`
	CreatedAt         time.Time `json:"created_at"`
}

type Member struct {
	ID          uuid.UUID `json:"id"`
	HouseholdID uuid.UUID `json:"household_id"`
	Name        string    `json:"name"`
	NameTh      string    `json:"name_th"`
	CreatedAt   time.Time `json:"created_at"`
}

type AccountType string

const (
	AccountJointChecking AccountType = "joint_checking"
	AccountJointCredit   AccountType = "joint_credit"
	AccountPersonal      AccountType = "personal"
)

type Account struct {
	ID          uuid.UUID   `json:"id"`
	HouseholdID uuid.UUID   `json:"household_id"`
	MemberID    *uuid.UUID  `json:"member_id,omitempty"`
	Name        string      `json:"name"`
	Type        AccountType `json:"type"`
	Institution string      `json:"institution,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type Category struct {
	ID          uuid.UUID  `json:"id"`
	HouseholdID uuid.UUID  `json:"household_id"`
	Name        string     `json:"name"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Transaction struct {
	ID              uuid.UUID  `json:"id"`
	HouseholdID     uuid.UUID  `json:"household_id"`
	AccountID       *uuid.UUID `json:"account_id,omitempty"`
	CategoryID      *uuid.UUID `json:"category_id,omitempty"`
	PaidByMemberID  *uuid.UUID `json:"paid_by_member_id,omitempty"`
	IsJoint         bool       `json:"is_joint"`
	TransactionDate time.Time  `json:"transaction_date"`
	Description     string     `json:"description"`
	AmountSatang    int64      `json:"amount_satang"`
	ReceiptPath     *string    `json:"receipt_path,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Splits          []Split    `json:"splits,omitempty"`
}

type AuthUser struct {
	LineUserID  string     `json:"line_user_id"`
	HouseholdID uuid.UUID  `json:"household_id"`
	MemberID    *uuid.UUID `json:"member_id,omitempty"`
	DisplayName string     `json:"display_name"`
	NeedsLink   bool       `json:"needs_link"`
}

type HouseholdSettings struct {
	OpenAIAPIKey           string `json:"openai_api_key,omitempty"`
	OpenAIModel            string `json:"openai_model"`
	LineChannelSecret      string `json:"line_channel_secret,omitempty"`
	LineChannelAccessToken string `json:"line_channel_access_token,omitempty"`
	HasOpenAIKey           bool   `json:"has_openai_key"`
	HasLineSecret          bool   `json:"has_line_secret"`
	HasLineToken           bool   `json:"has_line_token"`
}

type LineIdentity struct {
	LineUserID  string     `json:"line_user_id"`
	HouseholdID uuid.UUID  `json:"household_id"`
	MemberID    *uuid.UUID `json:"member_id,omitempty"`
	DisplayName string     `json:"display_name"`
}

type Caregiver struct {
	ID               uuid.UUID `json:"id"`
	HouseholdID      uuid.UUID `json:"household_id"`
	Name             string    `json:"name"`
	HourlyRateSatang int64     `json:"hourly_rate_satang"`
	DefaultStartTime *string   `json:"default_start_time,omitempty"`
	DefaultEndTime   *string   `json:"default_end_time,omitempty"`
	DefaultWeekdays  []int     `json:"default_weekdays"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"created_at"`
}

type PaidHoliday struct {
	ID          uuid.UUID `json:"id"`
	HouseholdID uuid.UUID `json:"household_id"`
	Date        string    `json:"date"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
}

type CaregiverHourEntry struct {
	ID          uuid.UUID  `json:"id"`
	CaregiverID uuid.UUID  `json:"caregiver_id"`
	HouseholdID uuid.UUID  `json:"household_id"`
	WorkDate    time.Time  `json:"work_date"`
	StartTime   *string    `json:"start_time,omitempty"`
	EndTime     *string    `json:"end_time,omitempty"`
	Hours       float64    `json:"hours"`
	Kind        string     `json:"kind"`
	Note        string     `json:"note"`
	CreatedAt   time.Time  `json:"created_at"`
}

type List struct {
	ID          uuid.UUID `json:"id"`
	HouseholdID uuid.UUID `json:"household_id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	Items       []ListItem `json:"items,omitempty"`
}

type ListItem struct {
	ID        uuid.UUID `json:"id"`
	ListID    uuid.UUID `json:"list_id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type PantryItem struct {
	ID          uuid.UUID  `json:"id"`
	HouseholdID uuid.UUID  `json:"household_id"`
	Name        string     `json:"name"`
	ExpiresOn   *time.Time `json:"expires_on,omitempty"`
	Notes       string     `json:"notes"`
	RebuyHint   string     `json:"rebuy_hint"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ReceiptParse struct {
	AmountSatang   *int64     `json:"amount_satang,omitempty"`
	Date           *string    `json:"date,omitempty"`
	Merchant       *string    `json:"merchant,omitempty"`
	AccountHolder  string     `json:"account_holder,omitempty"`
	PaidByMemberID *uuid.UUID `json:"paid_by_member_id,omitempty"`
	IsJoint        bool       `json:"is_joint"`
	RawText        string     `json:"raw_text,omitempty"`
}

type Split struct {
	ID           uuid.UUID `json:"id,omitempty"`
	TransactionID uuid.UUID `json:"transaction_id,omitempty"`
	MemberID     uuid.UUID `json:"member_id"`
	AmountSatang int64     `json:"amount_satang"`
}

type BalanceSnapshot struct {
	ID            uuid.UUID `json:"id"`
	AccountID     uuid.UUID `json:"account_id"`
	SnapshotDate  time.Time `json:"snapshot_date"`
	BalanceSatang int64     `json:"balance_satang"`
	Note          *string   `json:"note,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type CCStatementBalance struct {
	ID            uuid.UUID `json:"id"`
	AccountID     uuid.UUID `json:"account_id"`
	StatementDate time.Time `json:"statement_date"`
	BalanceSatang int64     `json:"balance_satang"`
	Note          *string   `json:"note,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Settlement struct {
	ID             uuid.UUID `json:"id"`
	HouseholdID    uuid.UUID `json:"household_id"`
	FromMemberID   uuid.UUID `json:"from_member_id"`
	ToMemberID     uuid.UUID `json:"to_member_id"`
	SettlementDate time.Time `json:"settlement_date"`
	AmountSatang   int64     `json:"amount_satang"`
	Note           *string   `json:"note,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type SavingsGoal struct {
	ID           uuid.UUID  `json:"id"`
	HouseholdID  uuid.UUID  `json:"household_id"`
	Name         string     `json:"name"`
	TargetSatang int64      `json:"target_satang"`
	CurrentSatang int64     `json:"current_satang"`
	Deadline     *time.Time `json:"deadline,omitempty"`
	CategoryID   *uuid.UUID `json:"category_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Contribution struct {
	ID                uuid.UUID `json:"id"`
	HouseholdID       uuid.UUID `json:"household_id"`
	MemberID          uuid.UUID `json:"member_id"`
	AccountID         uuid.UUID `json:"account_id"`
	ContributionMonth time.Time `json:"contribution_month"`
	AmountSatang      int64     `json:"amount_satang"`
	CreatedAt         time.Time `json:"created_at"`
}
