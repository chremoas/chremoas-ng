CREATE TABLE alliances
(
    id          INTEGER PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    ticker      VARCHAR(5) UNIQUE   NOT NULL,
    inserted_at TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP           NOT NULL DEFAULT NOW()
);

CREATE TABLE corporations
(
    id          INTEGER PRIMARY KEY,
    name        VARCHAR(100) UNIQUE NOT NULL,
    ticker      VARCHAR(5) UNIQUE   NOT NULL,
    alliance_id INTEGER REFERENCES alliances (id),
    inserted_at TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP           NOT NULL DEFAULT NOW()
);

CREATE TABLE characters
(
    id             INTEGER PRIMARY KEY,
    name           VARCHAR(100) UNIQUE NOT NULL,
    inserted_at    TIMESTAMP           NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP           NOT NULL DEFAULT NOW(),
    corporation_id INTEGER REFERENCES corporations (id),
    token          VARCHAR(255)        NOT NULL
);

CREATE TABLE authentication_codes
(
    character_id INTEGER REFERENCES characters (id),
    code         VARCHAR(20),
    used         BOOLEAN     NOT NULL DEFAULT FALSE,
    PRIMARY KEY (character_id, code)
);

CREATE TABLE user_character_map
(
    chat_id   VARCHAR(255) NOT NULL,
    character_id INTEGER REFERENCES characters (id),
    PRIMARY KEY (chat_id, character_id)
);
