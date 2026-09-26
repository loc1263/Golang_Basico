CREATE TABLE libros (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    titulo    TEXT    NOT NULL,
    autor_id  INTEGER NOT NULL REFERENCES autores(id),
    anio      INTEGER
);
CREATE INDEX idx_libros_autor ON libros(autor_id);
