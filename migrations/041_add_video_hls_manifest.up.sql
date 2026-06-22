-- Add HLS manifest support to videos. After a transcode worker finishes it
-- populates this column; the player switches from progressive MP4 to
-- adaptive HLS playback when present.
ALTER TABLE videos
    ADD COLUMN IF NOT EXISTS hls_manifest_key TEXT,
    ADD COLUMN IF NOT EXISTS transcoded_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_videos_hls
    ON videos(course_id)
    WHERE hls_manifest_key IS NOT NULL;