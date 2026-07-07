-- Schema for the shop domain (member / item / order), per example/spec.md and the ERD.
-- Loaded by deploy/compose.yaml into MySQL on first boot.

SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS member (
    id      BIGINT       NOT NULL AUTO_INCREMENT,
    name    VARCHAR(255) NOT NULL,
    city    VARCHAR(255),
    street  VARCHAR(255),
    zipcode VARCHAR(255),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS delivery (
    id      BIGINT      NOT NULL AUTO_INCREMENT,
    status  VARCHAR(20) NOT NULL, -- READY | COMP
    city    VARCHAR(255),
    street  VARCHAR(255),
    zipcode VARCHAR(255),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Single-table inheritance: dtype selects which type-specific columns apply.
CREATE TABLE IF NOT EXISTS item (
    id             BIGINT       NOT NULL AUTO_INCREMENT,
    name           VARCHAR(255) NOT NULL,
    price          INT          NOT NULL,
    stock_quantity INT          NOT NULL,
    dtype          VARCHAR(20)  NOT NULL, -- BOOK | ALBUM | MOVIE
    -- book
    author         VARCHAR(255),
    isbn           VARCHAR(255),
    -- album
    artist         VARCHAR(255),
    etc            VARCHAR(255),
    -- movie
    director       VARCHAR(255),
    actor          VARCHAR(255),
    created_at     DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_item_price (price),
    INDEX idx_item_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS category (
    id        BIGINT       NOT NULL AUTO_INCREMENT,
    parent_id BIGINT       NULL,
    name      VARCHAR(255) NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_category_parent FOREIGN KEY (parent_id) REFERENCES category (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS category_item (
    category_id BIGINT NOT NULL,
    item_id     BIGINT NOT NULL,
    PRIMARY KEY (category_id, item_id),
    CONSTRAINT fk_ci_category FOREIGN KEY (category_id) REFERENCES category (id),
    CONSTRAINT fk_ci_item     FOREIGN KEY (item_id)     REFERENCES item (id),
    INDEX idx_ci_item (item_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS orders (
    id          BIGINT      NOT NULL AUTO_INCREMENT,
    member_id   BIGINT      NOT NULL,
    delivery_id BIGINT      NULL,
    order_date  DATETIME    NOT NULL,
    status      VARCHAR(20) NOT NULL, -- ORDER | CANCEL
    PRIMARY KEY (id),
    CONSTRAINT fk_orders_member   FOREIGN KEY (member_id)   REFERENCES member (id),
    CONSTRAINT fk_orders_delivery FOREIGN KEY (delivery_id) REFERENCES delivery (id),
    INDEX idx_orders_member (member_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS order_item (
    id          BIGINT NOT NULL AUTO_INCREMENT,
    order_id    BIGINT NOT NULL,
    item_id     BIGINT NOT NULL,
    order_price INT    NOT NULL,
    count       INT    NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES orders (id),
    CONSTRAINT fk_oi_item  FOREIGN KEY (item_id)  REFERENCES item (id),
    INDEX idx_oi_order (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Minimal seed categories (optional; helps manual testing / E2E).
INSERT INTO category (id, parent_id, name) VALUES
    (1, NULL, 'ROOT'),
    (2, 1, 'BOOK'),
    (3, 1, 'ALBUM'),
    (4, 1, 'MOVIE');
