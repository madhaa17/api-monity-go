-- Daily closing prices per market symbol, recorded by the backend's daily close job. One row per
-- symbol and day, shared by every asset holding the symbol; manual per-asset prices stay in
-- asset_price_histories.

CREATE TABLE IF NOT EXISTS market_daily_closes (
  asset_type  asset_type NOT NULL,
  symbol      TEXT NOT NULL,
  currency    VARCHAR(10) NOT NULL,
  date        DATE NOT NULL,
  close       DECIMAL(20, 8) NOT NULL,
  source      TEXT NOT NULL,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (asset_type, symbol, currency, date)
);
