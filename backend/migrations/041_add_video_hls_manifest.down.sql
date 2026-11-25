DROP INDEX IF EXISTS idx_videos_hls;
ALTER TABLE videos
    DROP COLUMN IF EXISTS transcoded_at,
    DROP COLUMN IF EXISTS hls_manifest_key;