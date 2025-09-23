-- Create databases
CREATE DATABASE event_processor;
CREATE DATABASE image_indexer;

-- Enable UUID extension
\c event_processor;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c image_indexer;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- For production environments, uncomment the following lines to create separate users:
-- CREATE USER event_processor_user WITH PASSWORD 'event_processor_password';
-- CREATE USER image_indexer_user WITH PASSWORD 'image_indexer_password';

-- Grant permissions to the respective users:
-- GRANT ALL PRIVILEGES ON DATABASE event_processor TO event_processor_user;
-- GRANT ALL PRIVILEGES ON DATABASE image_indexer TO image_indexer_user;

-- Note: For development, all services use the same postgres user credentials
-- For production, update the docker-compose.yml environment variables to use:
-- NFT_INDEXER_POSTGRES_USER=event_processor_user (for event-processor)
-- NFT_INDEXER_POSTGRES_USER=image_indexer_user (for image-indexer)
-- And set the corresponding passwords
