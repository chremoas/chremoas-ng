CREATE TABLE authentication_scopes
(
    id        BIGSERIAL PRIMARY KEY,
    name      VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE alliances
(
    id          BIGINT PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    ticker      VARCHAR(5) UNIQUE   NOT NULL,
    inserted_at TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP           NOT NULL DEFAULT NOW()
);

CREATE TABLE corporations
(
    id          BIGINT PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    ticker      VARCHAR(5) UNIQUE   NOT NULL,
    alliance_id BIGINT REFERENCES alliances (id),
    inserted_at TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP           NOT NULL DEFAULT NOW()
);

CREATE TABLE characters
(
    id             BIGINT PRIMARY KEY,
    name           VARCHAR(100) UNIQUE NOT NULL,
    inserted_at    TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP           NOT NULL DEFAULT NOW(),
    corporation_id BIGINT REFERENCES corporations (id),
    token          VARCHAR(255)        NOT NULL
);

CREATE TABLE authentication_codes
(
    character_id BIGINT REFERENCES characters (id),
    code         VARCHAR(20),
    used         BOOLEAN     NOT NULL DEFAULT FALSE,
    PRIMARY KEY (character_id, code)
);

CREATE TABLE user_character_map
(
    chat_id   VARCHAR(255) NOT NULL,
    character_id BIGINT REFERENCES characters (id),
    PRIMARY KEY (chat_id, character_id)
);

CREATE TABLE authentication_scope_character_map
(
    character_id BIGINT REFERENCES characters (id),
    scope_id     BIGINT REFERENCES authentication_scopes (id),
    PRIMARY KEY (character_id, scope_id)
);
