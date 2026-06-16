-- Create OTP records table
CREATE TABLE otp_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    otp_hash VARCHAR(255) NOT NULL,
    purpose VARCHAR(50) NOT NULL CHECK (purpose IN ('registration', 'admin_login', 'email_change')),
    attempts INT DEFAULT 0,
    resend_count INT DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    invalidated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_otp_records_user_id ON otp_records(user_id);
CREATE INDEX idx_otp_records_expires_at ON otp_records(expires_at);
