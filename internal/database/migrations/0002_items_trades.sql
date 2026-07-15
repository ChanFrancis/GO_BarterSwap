CREATE TABLE items (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL,
    condition TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'disponible',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX items_category_idx ON items (category);
CREATE INDEX items_owner_idx ON items (owner_id);

CREATE TABLE trade_offers (
    id BIGSERIAL PRIMARY KEY,
    proposer_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    requested_item_id BIGINT NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    message TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'en_attente',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at TIMESTAMPTZ
);

CREATE INDEX trade_offers_proposer_idx ON trade_offers (proposer_id);
CREATE INDEX trade_offers_requested_item_idx ON trade_offers (requested_item_id);

-- Objets proposés en échange dans une offre (troc N objets contre 1)
CREATE TABLE trade_offer_items (
    offer_id BIGINT NOT NULL REFERENCES trade_offers (id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    PRIMARY KEY (offer_id, item_id)
);
