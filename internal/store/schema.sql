-- Schéma BarterSwap, appliqué au démarrage (idempotent).

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    pseudo TEXT NOT NULL,
    bio TEXT NOT NULL DEFAULT '',
    ville TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skills (
    user_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    nom TEXT NOT NULL,
    niveau TEXT NOT NULL,
    PRIMARY KEY (user_id, nom)
);

CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    provider_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    titre TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    categorie TEXT NOT NULL,
    duree_minutes INT NOT NULL,
    credits INT NOT NULL,
    ville TEXT NOT NULL DEFAULT '',
    actif BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS exchanges (
    id SERIAL PRIMARY KEY,
    service_id INT NOT NULL REFERENCES services (id) ON DELETE CASCADE,
    requester_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    owner_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Journal de transactions : le solde d'un utilisateur est la somme de ses
-- montants (positif = crédit, négatif = débit). Jamais de solde stocké.
CREATE TABLE IF NOT EXISTS credit_transactions (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- ON DELETE SET NULL : supprimer un échange (ou le service parent) ne doit
    -- pas effacer les écritures du journal, sinon les soldes seraient faussés.
    exchange_id INT REFERENCES exchanges (id) ON DELETE SET NULL,
    montant INT NOT NULL,
    type TEXT NOT NULL, -- earn, spend, refund
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reviews (
    id SERIAL PRIMARY KEY,
    exchange_id INT NOT NULL REFERENCES exchanges (id) ON DELETE CASCADE,
    author_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_id INT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    note INT NOT NULL CHECK (note BETWEEN 1 AND 5),
    commentaire TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (exchange_id, author_id) -- un seul avis par utilisateur et par échange
);
