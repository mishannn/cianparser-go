-- +goose Up
CREATE TABLE flat_median_price
(
    date_time DateTime,
    category String,
    rooms_count UInt8,
    location String,
    price_per_meter Float32
) ENGINE = MergeTree()
ORDER BY (date_time, category, rooms_count, location);

-- +goose Down
DROP TABLE flat_median_price;