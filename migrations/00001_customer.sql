-- +goose Up
CREATE TABLE member (
    id      BIGINT       NOT NULL AUTO_INCREMENT,
    name    VARCHAR(255) NOT NULL,
    city    VARCHAR(255),
    street  VARCHAR(255),
    zipcode VARCHAR(255),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE member;
