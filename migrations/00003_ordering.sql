-- +goose Up
CREATE TABLE delivery (
    id      BIGINT      NOT NULL AUTO_INCREMENT,
    status  VARCHAR(20) NOT NULL, -- READY | COMP
    city    VARCHAR(255),
    street  VARCHAR(255),
    zipcode VARCHAR(255),
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE orders (
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

CREATE TABLE order_item (
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

-- +goose Down
DROP TABLE order_item;
DROP TABLE orders;
DROP TABLE delivery;
