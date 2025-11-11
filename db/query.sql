-- name: CreateBook :one
INSERT INTO rag.book (book_name, book_text)
VALUES (
    $1, $2
)
RETURNING *;

-- name: ListBooks :many
SELECT
    id,
    book_name
FROM rag.book;

-- name: CreateBookPassages :batchexec
INSERT INTO rag.book_passage (book_id, passage_text, embedding)
VALUES (
    $1, $2, $3
);

-- name: QueryBook :many
SELECT
    id,
    passage_text,
    CAST(1 - (embedding <=> $2) AS REAL) AS similarity
FROM rag.book_passage
WHERE book_id = $1
ORDER BY embedding <=> $2
LIMIT $3;

-- name: BookExists :one
SELECT EXISTS(
    SELECT 1 FROM rag.book
    WHERE id = $1
);

-- name: GetBookPassages :many
SELECT
    id,
    book_id,
    passage_text
FROM rag.book_passage
WHERE book_id = $1;

-- name: GetAllBookPassages :many
SELECT
    id,
    book_id,
    passage_text
FROM rag.book_passage;
