-- +goose Up
CREATE TABLE category (
    id        BIGINT       NOT NULL AUTO_INCREMENT,
    parent_id BIGINT       NULL,
    name      VARCHAR(255) NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_category_parent FOREIGN KEY (parent_id) REFERENCES category (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE item (
    id             BIGINT       NOT NULL AUTO_INCREMENT,
    name           VARCHAR(255) NOT NULL,
    price          INT          NOT NULL,
    stock_quantity INT          NOT NULL,
    dtype          VARCHAR(20)  NOT NULL, -- BOOK | ALBUM | MOVIE
    author         VARCHAR(255),
    isbn           VARCHAR(255),
    artist         VARCHAR(255),
    etc            VARCHAR(255),
    director       VARCHAR(255),
    actor          VARCHAR(255),
    created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_item_price (price),
    INDEX idx_item_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE category_item (
    category_id BIGINT NOT NULL,
    item_id     BIGINT NOT NULL,
    PRIMARY KEY (category_id, item_id),
    CONSTRAINT fk_ci_category FOREIGN KEY (category_id) REFERENCES category (id),
    CONSTRAINT fk_ci_item     FOREIGN KEY (item_id)     REFERENCES item (id),
    INDEX idx_ci_item (item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO category (id, parent_id, name) VALUES
    (1, NULL, 'ROOT'),
    (2, 1, 'BOOK'),
    (3, 1, 'ALBUM'),
    (4, 1, 'MOVIE');

-- +goose Down
DROP TABLE category_item;
DROP TABLE item;
DROP TABLE category;
