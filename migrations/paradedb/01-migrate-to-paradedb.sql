-- Migration script: moving from PostgreSQL to ParadeDB
-- Note: back up the data before running this script

-- 1. Export the data (run against PostgreSQL)
-- pg_dump -U postgres -h localhost -p 5432 -d your_database > backup.sql

-- 2. Import the data (run against ParadeDB)
-- psql -U postgres -h localhost -p 5432 -d your_database < backup.sql

-- 3. Verify the data


-- Insert some sample data
-- INSERT INTO tenants (id, name, description, status, api_key)
-- VALUES 
--     (1, 'Demo Tenant', 'This is a demo tenant for testing', 'active', 'sk-00000001abcdefg123456')
-- ON CONFLICT DO NOTHING;

-- SELECT setval('tenants_id_seq', (SELECT MAX(id) FROM tenants));


-- -- Create knowledge base
-- INSERT INTO knowledge_bases (id, name, description, tenant_id, chunking_config, image_processing_config, embedding_model_id)
-- VALUES 
--     ('kb-00000001', 'Default Knowledge Base', 'Default knowledge base for testing', 1, '{"chunk_size": 512, "chunk_overlap": 50, "separators": ["\n\n", "\n", "।", ". "], "keep_separator": true}', '{"enable_multimodal": false, "model_id": ""}', 'model-embedding-00000001'),
--     ('kb-00000002', 'Test Knowledge Base', 'Test knowledge base for development', 1, '{"chunk_size": 512, "chunk_overlap": 50, "separators": ["\n\n", "\n", "।", ". "], "keep_separator": true}', '{"enable_multimodal": false, "model_id": ""}', 'model-embedding-00000001'),
--     ('kb-00000003', 'Test Knowledge Base 2', 'Test knowledge base for development 2', 1, '{"chunk_size": 512, "chunk_overlap": 50, "separators": ["\n\n", "\n", "।", ". "], "keep_separator": true}', '{"enable_multimodal": false, "model_id": ""}', 'model-embedding-00000001')
-- ON CONFLICT DO NOTHING;


SELECT COUNT(*) FROM tenants;
SELECT COUNT(*) FROM models;
SELECT COUNT(*) FROM knowledge_bases;
SELECT COUNT(*) FROM knowledges;


-- Smoke test for BM25 full-text search

-- Create the document table
CREATE TABLE bm25_smoke_documents (
    id SERIAL PRIMARY KEY,
    title TEXT,
    content TEXT,
    published_date DATE
);

-- Create the BM25 index on the table. The default tokenizer covers English and
-- other space-delimited scripts, including Devanagari.
CREATE INDEX idx_documents_bm25 ON bm25_smoke_documents
USING bm25 (id, content)
WITH (
    key_field = 'id',
    text_fields = '{
        "content": {
          "tokenizer": {"type": "default"}
        }
    }'
);

INSERT INTO bm25_smoke_documents (title, content, published_date)
VALUES
('Leave policy', 'Employees accrue 18 days of earned leave each calendar year. Unused leave carries forward up to 45 days.', '2026-01-15'),
('Expense reimbursement', 'Claims above Rs 5,000 need a manager approval and the original GST invoice attached.', '2026-02-20'),
('GST basics', 'Goods and Services Tax replaced most indirect taxes in India and is filed monthly through the GSTN portal.', '2026-03-10'),
('कर्मचारी नीति', 'कर्मचारियों को हर वर्ष अठारह दिन का अर्जित अवकाश मिलता है।', '2026-04-05'),
('Product manual', 'The indexing pipeline chunks a document, embeds each chunk and writes both dense and sparse vectors.', '2026-05-12');

INSERT INTO bm25_smoke_documents (title, content, published_date)
VALUES
('hello world', 'hello world', '2026-05-12');
