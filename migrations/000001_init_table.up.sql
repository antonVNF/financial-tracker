CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    amount NUMERIC DEFAULT 0,
    category VARCHAR(100),
    description VARCHAR(200),
    date DATE,
    created_at TIMESTAMP DEFAULT NOW()
    );