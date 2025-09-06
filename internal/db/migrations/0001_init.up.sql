CREATE TABLE IF NOT EXISTS subscriptions (
    id              UUID    PRIMARY KEY,             
    service_name    TEXT    NOT NULL,
    user_id         UUID    NOT NULL, 
    price           INTEGER NOT NULL CHECK (price > 0),
    start_date      DATE    NOT NULL,
    end_date        DATE
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_service ON subscriptions(service_name);
