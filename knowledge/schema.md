model User {
  id        BigInt   @id @default(autoincrement())
  uuid      String   @unique @default(uuid())

  email     String   @unique
  password  String
  name      String?
  role      UserRole @default(USER)

  assets    Asset[]
  incomes   Income[]
  expenses  Expense[]
  savings   SavingGoal[]

  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  @@index([uuid])
}

model Asset {
  id        BigInt   @id @default(autoincrement())
  uuid      String   @unique @default(uuid())

  userId    BigInt
  user      User     @relation(fields: [userId], references: [id], onDelete: Cascade)

  name      String
  type      AssetType
  quantity  Decimal  @db.Decimal(20, 8)
  symbol    String?

  // Purchase Information
  purchasePrice    Decimal  @db.Decimal(20, 8) @default(0)
  purchaseDate     DateTime
  purchaseCurrency String   @default("USD")
  totalCost        Decimal  @db.Decimal(20, 8) @default(0)

  // Additional Costs (optional)
  transactionFee  Decimal? @db.Decimal(20, 8)
  maintenanceCost Decimal? @db.Decimal(20, 8)

  // Target & Planning (optional)
  targetPrice Decimal?  @db.Decimal(20, 8)
  targetDate  DateTime?

  // Real Asset Specific (optional)
  estimatedYield Decimal? @db.Decimal(20, 8)
  yieldPeriod    String?

  // Documentation (optional)
  description String?
  notes       String?

  // Status
  status    AssetStatus @default(ACTIVE)
  soldAt    DateTime?
  soldPrice Decimal?    @db.Decimal(20, 8)

  priceHistories AssetPriceHistory[]

  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt

  @@index([userId])
  @@index([uuid])
  @@index([status])
  @@index([purchaseDate])
}

model AssetPriceHistory {
  id        BigInt   @id @default(autoincrement())
  uuid      String   @unique @default(uuid())

  assetId  BigInt
  asset    Asset    @relation(fields: [assetId], references: [id], onDelete: Cascade)

  price     Decimal  @db.Decimal(20, 8)
  source    String
  recordedAt DateTime @default(now())

  @@index([assetId])
}

model Income {
  id        BigInt   @id @default(autoincrement())
  uuid      String   @unique @default(uuid())

  userId    BigInt
  user      User     @relation(fields: [userId], references: [id], onDelete: Cascade)

  amount    Decimal  @db.Decimal(20, 2)
  source    String
  note      String?
  date      DateTime

  createdAt DateTime @default(now())

  @@index([userId, date])
  @@index([uuid])
}

model Expense {
  id        BigInt   @id @default(autoincrement())
  uuid      String   @unique @default(uuid())

  userId    BigInt
  user      User     @relation(fields: [userId], references: [id], onDelete: Cascade)

  amount    Decimal  @db.Decimal(20, 2)
  category  ExpenseCategory
  note      String?
  date      DateTime

  createdAt DateTime @default(now())

  @@index([userId, date])
  @@index([uuid])
}

model SavingGoal {
  id        BigInt   @id @default(autoincrement())
  uuid      String   @unique @default(uuid())

  userId    BigInt
  user      User     @relation(fields: [userId], references: [id], onDelete: Cascade)

  title       String
  targetAmount Decimal @db.Decimal(20, 2)
  currentAmount Decimal @db.Decimal(20, 2) @default(0)
  deadline    DateTime?

  createdAt   DateTime @default(now())
  updatedAt   DateTime @updatedAt

  @@index([userId])
  @@index([uuid])
}
