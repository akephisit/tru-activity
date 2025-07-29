-- Migration to update subscription system and add analytics tables

-- Update subscription_type enum to include enterprise
ALTER TYPE subscription_type ADD VALUE 'enterprise';

-- Update subscriptions table structure
ALTER TABLE subscriptions 
    DROP COLUMN IF EXISTS user_id,
    ADD COLUMN IF NOT EXISTS faculty_id INTEGER NOT NULL REFERENCES faculties(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS notification_sent_7_days BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS notification_sent_1_day BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS last_notification_at TIMESTAMP WITH TIME ZONE;

-- Create index for faculty_id on subscriptions
CREATE INDEX IF NOT EXISTS idx_subscriptions_faculty_id ON subscriptions(faculty_id);

-- Create faculty_metrics table
CREATE TABLE IF NOT EXISTS faculty_metrics (
    id SERIAL PRIMARY KEY,
    faculty_id INTEGER NOT NULL REFERENCES faculties(id) ON DELETE CASCADE,
    total_students INTEGER DEFAULT 0,
    active_students INTEGER DEFAULT 0,
    total_activities INTEGER DEFAULT 0,
    completed_activities INTEGER DEFAULT 0,
    total_participants INTEGER DEFAULT 0,
    average_attendance DECIMAL(5,2) DEFAULT 0.00,
    date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create unique index on faculty_id and date for faculty_metrics
CREATE UNIQUE INDEX IF NOT EXISTS idx_faculty_metrics_faculty_date ON faculty_metrics(faculty_id, date);

-- Create system_metrics table
CREATE TABLE IF NOT EXISTS system_metrics (
    id SERIAL PRIMARY KEY,
    total_faculties INTEGER DEFAULT 0,
    total_departments INTEGER DEFAULT 0,
    total_students INTEGER DEFAULT 0,
    total_activities INTEGER DEFAULT 0,
    total_participations INTEGER DEFAULT 0,
    active_subscriptions INTEGER DEFAULT 0,
    expired_subscriptions INTEGER DEFAULT 0,
    date DATE NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create notification_logs table
CREATE TABLE IF NOT EXISTS notification_logs (
    id SERIAL PRIMARY KEY,
    subscription_id INTEGER NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('expiry_7_days', 'expiry_1_day', 'expired')),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed')),
    email VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for notification_logs
CREATE INDEX IF NOT EXISTS idx_notification_logs_subscription_id ON notification_logs(subscription_id);
CREATE INDEX IF NOT EXISTS idx_notification_logs_status ON notification_logs(status);
CREATE INDEX IF NOT EXISTS idx_notification_logs_type ON notification_logs(type);

-- Create function to automatically update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add triggers for updated_at
CREATE TRIGGER update_faculty_metrics_updated_at BEFORE UPDATE ON faculty_metrics 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_system_metrics_updated_at BEFORE UPDATE ON system_metrics 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_logs_updated_at BEFORE UPDATE ON notification_logs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data for testing (optional - remove in production)
-- Insert some initial faculty metrics
INSERT INTO faculty_metrics (faculty_id, total_students, active_students, total_activities, completed_activities, total_participants, average_attendance, date)
SELECT 
    f.id,
    0,
    0,
    0,
    0,
    0,
    0.00,
    CURRENT_DATE
FROM faculties f
WHERE NOT EXISTS (
    SELECT 1 FROM faculty_metrics fm 
    WHERE fm.faculty_id = f.id AND fm.date = CURRENT_DATE
);

-- Insert initial system metrics
INSERT INTO system_metrics (
    total_faculties, 
    total_departments, 
    total_students, 
    total_activities, 
    total_participations, 
    active_subscriptions, 
    expired_subscriptions, 
    date
)
SELECT 
    (SELECT COUNT(*) FROM faculties WHERE deleted_at IS NULL),
    (SELECT COUNT(*) FROM departments WHERE deleted_at IS NULL),
    (SELECT COUNT(*) FROM users WHERE role = 'student' AND deleted_at IS NULL),
    (SELECT COUNT(*) FROM activities WHERE deleted_at IS NULL),
    (SELECT COUNT(*) FROM participations WHERE deleted_at IS NULL),
    (SELECT COUNT(*) FROM subscriptions WHERE status = 'active' AND deleted_at IS NULL),
    (SELECT COUNT(*) FROM subscriptions WHERE status = 'expired' AND deleted_at IS NULL),
    CURRENT_DATE
WHERE NOT EXISTS (
    SELECT 1 FROM system_metrics WHERE date = CURRENT_DATE
);