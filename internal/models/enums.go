package models

type UserRole string

const (
	UserRoleUser  UserRole = "USER"
	UserRoleAdmin UserRole = "ADMIN"
)

type AssetType string

const (
	AssetTypeCrypto     AssetType = "CRYPTO"
	AssetTypeStock      AssetType = "STOCK"
	AssetTypeCash       AssetType = "CASH"
	AssetTypeRealEstate AssetType = "REAL_ESTATE"
	AssetTypeLivestock  AssetType = "LIVESTOCK"
	AssetTypeOther      AssetType = "OTHER"
)

type AssetStatus string

const (
	AssetStatusActive  AssetStatus = "ACTIVE"
	AssetStatusSold    AssetStatus = "SOLD"
	AssetStatusPlanned AssetStatus = "PLANNED"
)

type ObligationStatus string

const (
	ObligationStatusPending ObligationStatus = "PENDING"
	ObligationStatusPartial ObligationStatus = "PARTIAL"
	ObligationStatusPaid    ObligationStatus = "PAID"
	ObligationStatusOverdue ObligationStatus = "OVERDUE"
)
