ALTER TABLE gifs
    ADD COLUMN url           TEXT NOT NULL DEFAULT '',
    ADD COLUMN thumbnail_url TEXT DEFAULT '',
    ADD COLuMN name          TEXT DEFAULT '';