-- name: CreateBook
INSERT INTO rag.book (book_name, book_text)
VALUES (
    $1, $2
);

-- name: ListBooks
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
    CAST(1 - (embedding <=> $1) AS REAL) AS similarity
FROM rag.book_passage
WHERE book_id = $1
ORDER BY embedding <=> $2
LIMIT $3;
