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

type ExpenseCategory string

const (
	ExpenseCategoryFood     ExpenseCategory = "FOOD"
	ExpenseCategoryTransport ExpenseCategory = "TRANSPORT"
	ExpenseCategoryHousing  ExpenseCategory = "HOUSING"
	ExpenseCategoryUtilities ExpenseCategory = "UTILITIES"
	ExpenseCategoryHealth   ExpenseCategory = "HEALTH"
	ExpenseCategoryEntertainment ExpenseCategory = "ENTERTAINMENT"
	ExpenseCategoryShopping ExpenseCategory = "SHOPPING"
	ExpenseCategoryOther   ExpenseCategory = "OTHER"
)
