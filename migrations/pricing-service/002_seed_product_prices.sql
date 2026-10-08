INSERT INTO product_prices (
    product_id,
    amount_cents,
    currency,
    discount_percent
) VALUES
    ('42', 12990, 'BRL', 10),
    ('100', 8990, 'BRL', 0),
    ('keyboard-pro', 24990, 'BRL', 15)
ON CONFLICT (product_id) DO UPDATE SET
    amount_cents = EXCLUDED.amount_cents,
    currency = EXCLUDED.currency,
    discount_percent = EXCLUDED.discount_percent,
    updated_at = now();
